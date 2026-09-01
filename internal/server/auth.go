package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

type authSessionStore interface {
	Create(uuid.UUID, []byte, time.Time) (*models.ServerSession, error)
	GetActiveByTokenHash([]byte, time.Time) (*models.ServerSession, error)
	RevokeByTokenHash([]byte, time.Time) error
}

type authUserStore interface {
	GetByLogin(string) (*models.User, error)
	GetByID(uuid.UUID) (*models.User, error)
	UpdatePassword(uuid.UUID, string) error
	IncrementFailedLoginAttempts(uuid.UUID) (int, bool, error)
	ResetFailedLoginAttempts(uuid.UUID) error
}

type authSettingsStore interface {
	Get(string) (*models.SystemSetting, error)
}

type authContextKey struct{}

type authenticatedRequest struct {
	Session   *models.ServerSession
	User      *models.User
	TokenHash []byte
}

type initialSetupStore interface {
	CountUsers() (int, error)
	CreateInitialAdmin(string) error
}

type initialSetupRequest struct {
	Password string `json:"password"`
}

func (api *managementAPI) setupRequired(w http.ResponseWriter, _ *http.Request) {
	if api.initialSetup == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "initial_setup_unavailable", errors.New("initial setup store is not configured"))
		return
	}
	count, err := api.initialSetup.CountUsers()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "initial_setup_check_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"required": count == 0})
}

func (api *managementAPI) initialSetupAdmin(w http.ResponseWriter, r *http.Request) {
	if api.initialSetup == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "initial_setup_unavailable", errors.New("initial setup store is not configured"))
		return
	}
	count, err := api.initialSetup.CountUsers()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "initial_setup_check_failed", err)
		return
	}
	if count > 0 {
		writeUserError(w, models.NewConflict("начальная настройка уже выполнена"))
		return
	}
	var req initialSetupRequest
	if err := decodeJSON(r, &req); err != nil || req.Password == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", errors.New("password is required"))
		return
	}
	if err := security.ValidatePassword(req.Password); err != nil {
		writeUserError(w, models.NewBadRequestWrapped(err.Error(), err))
		return
	}
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "initial_setup_failed", err)
		return
	}
	if err := api.initialSetup.CreateInitialAdmin(hash); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	User        *dto.User `json:"user"`
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type changeRequiredPasswordRequest struct {
	Login       string `json:"login"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (api *managementAPI) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Login) == "" || req.Password == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", errors.New("login and password are required"))
		return
	}
	login := strings.TrimSpace(req.Login)
	authKey := remoteHost(r.RemoteAddr) + "\x00" + login
	if !api.authenticationAllowed(authKey, time.Now()) {
		writeAPIError(w, http.StatusTooManyRequests, "authentication_rate_limited", errors.New("too many authentication failures; retry later"))
		return
	}
	user, err := api.authUsers.GetByLogin(login)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "authentication_failed", err)
		return
	}
	if user == nil || !security.VerifyPassword(user.PasswordHash, req.Password) {
		api.recordAuthenticationFailure(authKey, time.Now())
		if user != nil {
			attempts, active, incrementErr := api.authUsers.IncrementFailedLoginAttempts(user.ID)
			if incrementErr != nil {
				writeAPIError(w, http.StatusInternalServerError, "authentication_failed", incrementErr)
				return
			}
			if attempts >= 5 || !active {
				api.auditAction(user, "USER_LOCKED", "Учётная запись заблокирована после 5 неверных попыток входа через docflow-server")
				writeAPIError(w, http.StatusForbidden, "user_locked", models.ErrUserLocked)
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
		writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", models.ErrInvalidCredentials)
		return
	}
	if !user.IsActive {
		code := "user_inactive"
		authErr := error(models.ErrUserNotActive)
		if user.FailedLoginAttempts >= 5 {
			code, authErr = "user_locked", models.ErrUserLocked
		}
		writeAPIError(w, http.StatusForbidden, code, authErr)
		return
	}
	if user.FailedLoginAttempts > 0 {
		if err := api.authUsers.ResetFailedLoginAttempts(user.ID); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "authentication_failed", err)
			return
		}
	}
	if api.passwordChangeRequired(user) {
		writeAPIError(w, http.StatusForbidden, "password_change_required", models.ErrPasswordChangeRequired)
		return
	}
	api.clearAuthenticationFailures(authKey)
	token, tokenHash, err := newSessionToken()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_creation_failed", err)
		return
	}
	expiresAt := time.Now().UTC().Add(time.Duration(api.cfg.Server.SessionTTLHours) * time.Hour)
	if _, err := api.sessions.Create(user.ID, tokenHash, expiresAt); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_creation_failed", err)
		return
	}
	api.auditAction(user, "LOGIN", "Вход через docflow-server")
	writeJSON(w, http.StatusOK, loginResponse{AccessToken: token, ExpiresAt: expiresAt, User: dto.MapUser(user)})
}

func (api *managementAPI) logout(w http.ResponseWriter, r *http.Request) {
	auth := authenticatedFromContext(r.Context())
	if err := api.sessions.RevokeByTokenHash(auth.TokenHash, time.Now().UTC()); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "logout_failed", err)
		return
	}
	api.auditAction(auth.User, "LOGOUT", "Выход через docflow-server")
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, dto.MapUser(authenticatedFromContext(r.Context()).User))
}

func (api *managementAPI) changePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil || req.OldPassword == "" || req.NewPassword == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", errors.New("oldPassword and newPassword are required"))
		return
	}
	auth := authenticatedFromContext(r.Context())
	if !security.VerifyPassword(auth.User.PasswordHash, req.OldPassword) {
		writeAPIError(w, http.StatusBadRequest, "wrong_password", models.ErrWrongPassword)
		return
	}
	if err := api.updatePassword(auth.User, req.NewPassword); err != nil {
		api.writePasswordUpdateError(w, err)
		return
	}
	api.auditAction(auth.User, "USER_PASSWORD_CHANGED", "Пользователь изменил пароль через docflow-server; все активные сессии отозваны")
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) changeRequiredPassword(w http.ResponseWriter, r *http.Request) {
	var req changeRequiredPasswordRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Login) == "" || req.OldPassword == "" || req.NewPassword == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", errors.New("login, oldPassword and newPassword are required"))
		return
	}
	login := strings.TrimSpace(req.Login)
	authKey := remoteHost(r.RemoteAddr) + "\x00" + login
	if !api.authenticationAllowed(authKey, time.Now()) {
		writeAPIError(w, http.StatusTooManyRequests, "authentication_rate_limited", errors.New("too many authentication failures; retry later"))
		return
	}
	user, err := api.authUsers.GetByLogin(login)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "password_change_failed", err)
		return
	}
	if user == nil || !security.VerifyPassword(user.PasswordHash, req.OldPassword) {
		api.recordAuthenticationFailure(authKey, time.Now())
		if user != nil {
			attempts, active, incrementErr := api.authUsers.IncrementFailedLoginAttempts(user.ID)
			if incrementErr != nil {
				writeAPIError(w, http.StatusInternalServerError, "password_change_failed", incrementErr)
				return
			}
			if attempts >= 5 || !active {
				api.auditAction(user, "USER_LOCKED", "Учётная запись заблокирована после 5 неверных попыток обязательной смены пароля")
				writeAPIError(w, http.StatusForbidden, "user_locked", models.ErrUserLocked)
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
		writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", models.ErrInvalidCredentials)
		return
	}
	if !user.IsActive {
		code, authErr := "user_inactive", error(models.ErrUserNotActive)
		if user.FailedLoginAttempts >= 5 {
			code, authErr = "user_locked", models.ErrUserLocked
		}
		writeAPIError(w, http.StatusForbidden, code, authErr)
		return
	}
	if !api.passwordChangeRequired(user) {
		writeAPIError(w, http.StatusConflict, "password_change_not_required", models.NewConflict("смена пароля сейчас не требуется"))
		return
	}
	if err := api.updatePassword(user, req.NewPassword); err != nil {
		api.writePasswordUpdateError(w, err)
		return
	}
	api.clearAuthenticationFailures(authKey)
	api.auditAction(user, "USER_PASSWORD_CHANGED", "Выполнена обязательная смена пароля через docflow-server; все активные сессии отозваны")
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) updatePassword(user *models.User, newPassword string) error {
	if err := security.ValidatePassword(newPassword); err != nil {
		return models.NewBadRequestWrapped(err.Error(), err)
	}
	newHash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return api.authUsers.UpdatePassword(user.ID, newHash)
}

func (api *managementAPI) writePasswordUpdateError(w http.ResponseWriter, err error) {
	if appErr, ok := models.AsAppError(err); ok && appErr.StatusCode() < http.StatusInternalServerError {
		writeAPIError(w, appErr.StatusCode(), strings.ToLower(appErr.SafeKind()), appErr)
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "password_change_failed", err)
}

func (api *managementAPI) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "authentication_required", models.ErrUnauthorized)
			return
		}
		hash := sha256.Sum256([]byte(token))
		session, err := api.sessions.GetActiveByTokenHash(hash[:], time.Now().UTC())
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "session_validation_failed", err)
			return
		}
		if session == nil {
			writeAPIError(w, http.StatusUnauthorized, "session_invalid", models.ErrUnauthorized)
			return
		}
		user, err := api.authUsers.GetByID(session.UserID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "session_validation_failed", err)
			return
		}
		if user == nil || !user.IsActive {
			_ = api.sessions.RevokeByTokenHash(hash[:], time.Now().UTC())
			writeAPIError(w, http.StatusUnauthorized, "session_invalid", models.ErrUnauthorized)
			return
		}
		auth := &authenticatedRequest{Session: session, User: user, TokenHash: append([]byte(nil), hash[:]...)}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authContextKey{}, auth)))
	})
}

func (api *managementAPI) requirePermission(permission string, next http.Handler) http.Handler {
	return api.requireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := authenticatedFromContext(r.Context())
		if auth == nil || auth.User == nil || !contains(auth.User.SystemPermissions, permission) {
			writeAPIError(w, http.StatusForbidden, "forbidden", models.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func authenticatedFromContext(ctx context.Context) *authenticatedRequest {
	auth, _ := ctx.Value(authContextKey{}).(*authenticatedRequest)
	return auth
}

func bearerToken(value string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(value), " ")
	return token, ok && strings.EqualFold(scheme, "Bearer") && token != ""
}

func newSessionToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], nil
}

func (api *managementAPI) passwordChangeRequired(user *models.User) bool {
	if user.PasswordChangeRequired {
		return true
	}
	setting, err := api.authSettings.Get("password_lifetime_days")
	if err != nil || setting == nil {
		return false
	}
	days, err := strconv.Atoi(strings.TrimSpace(setting.Value))
	if err != nil || days <= 0 {
		return false
	}
	return user.PasswordChangedAt == nil || !time.Now().Before(user.PasswordChangedAt.AddDate(0, 0, days))
}

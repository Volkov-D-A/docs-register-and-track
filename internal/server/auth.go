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

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	User        *dto.User `json:"user"`
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

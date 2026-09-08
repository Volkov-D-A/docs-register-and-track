package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type scenarioAuthUsers struct {
	fakeAuthUsers
	lookupErr          error
	increments, resets int
}

func (s *scenarioAuthUsers) GetByLogin(login string) (*models.User, error) {
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	return s.fakeAuthUsers.GetByLogin(login)
}
func (s *scenarioAuthUsers) IncrementFailedLoginAttempts(uuid.UUID) (int, bool, error) {
	s.increments++
	s.user.FailedLoginAttempts++
	if s.user.FailedLoginAttempts >= 5 {
		s.user.IsActive = false
	}
	return s.user.FailedLoginAttempts, s.user.IsActive, nil
}
func (s *scenarioAuthUsers) ResetFailedLoginAttempts(uuid.UUID) error {
	s.resets++
	s.user.FailedLoginAttempts = 0
	return nil
}

type scenarioAuthSettings struct{ value string }

func (s scenarioAuthSettings) Get(string) (*models.SystemSetting, error) {
	return &models.SystemSetting{Value: s.value}, nil
}

func TestServerLoginCredentialScenarios(t *testing.T) {
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	expired := time.Now().Add(-48 * time.Hour)
	for _, tc := range []struct {
		name, login, password, code          string
		active, required                     bool
		attempts, status, increments, resets int
		changed                              *time.Time
		lookupErr                            error
	}{
		{name: "unknown", login: "missing", password: "Passw0rd!", active: true, status: 401, code: "invalid_credentials"},
		{name: "wrong password", password: "wrong", active: true, status: 401, code: "invalid_credentials", increments: 1},
		{name: "fifth failure", password: "wrong", active: true, attempts: 4, status: 403, code: "user_locked", increments: 1},
		{name: "inactive", password: "Passw0rd!", status: 403, code: "user_inactive"},
		{name: "locked", password: "Passw0rd!", attempts: 5, status: 403, code: "user_locked"},
		{name: "required", password: "Passw0rd!", active: true, required: true, status: 403, code: "password_change_required"},
		{name: "expired", password: "Passw0rd!", active: true, changed: &expired, status: 403, code: "password_change_required"},
		{name: "reset failures", password: "Passw0rd!", active: true, attempts: 3, status: 200, resets: 1},
		{name: "storage failure", password: "Passw0rd!", active: true, status: 500, code: "authentication_failed", lookupErr: errors.New("store unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			users := &scenarioAuthUsers{fakeAuthUsers: fakeAuthUsers{user: &models.User{ID: uuid.New(), Login: "user", PasswordHash: hash, IsActive: tc.active, FailedLoginAttempts: tc.attempts, PasswordChangeRequired: tc.required, PasswordChangedAt: tc.changed}}, lookupErr: tc.lookupErr}
			sessions := &fakeAuthSessions{}
			api := &managementAPI{cfg: validConfig(), authUsers: users, authSettings: fakeAuthSettings{}, sessions: sessions, audit: &fakeAdminAudit{}}
			if tc.changed != nil {
				api.authSettings = scenarioAuthSettings{value: "1"}
			}
			login := tc.login
			if login == "" {
				login = "user"
			}
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"`+login+`","password":"`+tc.password+`"}`)))
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.code != "" {
				require.Contains(t, response.Body.String(), `"code":"`+tc.code+`"`)
				require.Nil(t, sessions.session)
			} else {
				require.NotNil(t, sessions.session)
			}
			require.Equal(t, tc.increments, users.increments)
			require.Equal(t, tc.resets, users.resets)
		})
	}
}

func TestServerRequiredPasswordRejectsInvalidChanges(t *testing.T) {
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	for _, tc := range []struct {
		name, old, new     string
		required           bool
		status, increments int
	}{
		{"wrong old", "wrong", "NewPassw0rd!", true, 401, 1},
		{"weak new", "Passw0rd!", "123", true, 400, 0},
		{"not required", "Passw0rd!", "NewPassw0rd!", false, 409, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			users := &scenarioAuthUsers{fakeAuthUsers: fakeAuthUsers{user: &models.User{ID: uuid.New(), Login: "user", PasswordHash: hash, IsActive: true, PasswordChangeRequired: tc.required}}}
			api := &managementAPI{authUsers: users, authSettings: fakeAuthSettings{}, audit: &fakeAdminAudit{}}
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-required-password", strings.NewReader(`{"login":"user","oldPassword":"`+tc.old+`","newPassword":"`+tc.new+`"}`)))
			require.Equal(t, tc.status, response.Code, response.Body.String())
			require.Equal(t, hash, users.user.PasswordHash)
			require.Equal(t, tc.increments, users.increments)
		})
	}
}

func TestServerSessionRechecksUserAndPermissions(t *testing.T) {
	for _, change := range []string{"removed", "inactive", "permissions revoked"} {
		t.Run(change, func(t *testing.T) {
			api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
			users := api.authUsers.(*fakeAuthUsers)
			switch change {
			case "removed":
				users.user = nil
			case "inactive":
				users.user.IsActive = false
			case "permissions revoked":
				users.user.SystemPermissions = nil
			}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			api.Handler().ServeHTTP(response, request)
			status := http.StatusUnauthorized
			if change == "permissions revoked" {
				status = http.StatusForbidden
			}
			require.Equal(t, status, response.Code, response.Body.String())
		})
	}
}

func TestServerPasswordAndProfileRequireSession(t *testing.T) {
	for _, path := range []string{"/api/v1/auth/change-password", "/api/v1/profile"} {
		method := http.MethodPost
		if path == "/api/v1/profile" {
			method = http.MethodPatch
		}
		response := httptest.NewRecorder()
		api := &managementAPI{}
		api.Handler().ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(`{}`)))
		require.Equal(t, http.StatusUnauthorized, response.Code)
	}
}

func TestServerPasswordRejectsWrongOldAndWeakNew(t *testing.T) {
	for _, body := range []string{`{"oldPassword":"wrong","newPassword":"NewPassw0rd!"}`, `{"oldPassword":"Passw0rd!","newPassword":"123"}`} {
		api, _, token := authenticatedUserAPI(t, nil)
		hash := api.authUsers.(*fakeAuthUsers).user.PasswordHash
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		require.Equal(t, hash, api.authUsers.(*fakeAuthUsers).user.PasswordHash)
	}
}

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/security"
)

type fakeAuthUsers struct{ user *models.User }

func TestDummyLoginPasswordHashMatchesPasswordCost(t *testing.T) {
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	wantCost, err := bcrypt.Cost([]byte(hash))
	require.NoError(t, err)
	gotCost, err := bcrypt.Cost([]byte(dummyLoginPasswordHash))
	require.NoError(t, err)
	require.Equal(t, wantCost, gotCost)
	require.True(t, security.VerifyPassword(dummyLoginPasswordHash, "docflow-dummy-password"))
}

func TestAuthenticationRoutesVerifyPasswordOnceForUnknownAndKnownUsers(t *testing.T) {
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	user := &models.User{ID: uuid.New(), Login: "user", PasswordHash: hash, IsActive: true}
	for _, tc := range []struct {
		name     string
		user     *models.User
		password string
		wantHash string
	}{
		{"unknown user", nil, "wrong-password", dummyLoginPasswordHash},
		{"known user", user, "wrong-password", hash},
		{"matching dummy password", nil, "docflow-dummy-password", dummyLoginPasswordHash},
	} {
		for _, route := range []string{"login", "change-required-password", "migration"} {
			t.Run(route+"/"+tc.name, func(t *testing.T) {
				calls := 0
				sessions := &fakeAuthSessions{}
				api := &managementAPI{
					users:     fakeAdminUsers{user: tc.user},
					authUsers: &fakeAuthUsers{user: tc.user},
					sessions:  sessions,
					verifyPasswordHash: func(gotHash, password string) bool {
						calls++
						assert.Equal(t, tc.wantHash, gotHash)
						assert.Equal(t, tc.password, password)
						return security.VerifyPassword(gotHash, password)
					},
				}
				var input any = loginRequest{Login: "user", Password: tc.password}
				if route == "change-required-password" {
					input = changeRequiredPasswordRequest{Login: "user", OldPassword: tc.password, NewPassword: "NewPassw0rd!"}
				}
				body, err := json.Marshal(input)
				require.NoError(t, err)
				response := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/"+route, bytes.NewReader(body))
				if route == "migration" {
					request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/migrations/apply", nil)
					request.SetBasicAuth("user", tc.password)
				}
				api.Handler().ServeHTTP(response, request)
				require.Equal(t, 1, calls)
				require.Equal(t, http.StatusUnauthorized, response.Code)
				var result map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				assert.Equal(t, "invalid_credentials", result["code"])
				if route != "migration" {
					assert.Equal(t, "неверный логин или пароль", result["error"])
				}
				assert.Nil(t, sessions.session)
			})
		}
	}
}

func (f *fakeAuthUsers) GetByLogin(login string) (*models.User, error) {
	if f.user != nil && f.user.Login == login {
		return f.user, nil
	}
	return nil, nil
}
func (f *fakeAuthUsers) GetByID(id uuid.UUID) (*models.User, error) {
	if f.user != nil && f.user.ID == id {
		return f.user, nil
	}
	return nil, nil
}
func (f *fakeAuthUsers) IncrementFailedLoginAttemptsWithOutbox(uuid.UUID, models.OutboxEvent) (int, bool, error) {
	return 1, true, nil
}
func (f *fakeAuthUsers) ResetFailedLoginAttempts(uuid.UUID) error { return nil }
func (f *fakeAuthUsers) UpdatePassword(id uuid.UUID, passwordHash string) error {
	if f.user == nil || f.user.ID != id {
		return models.NewNotFound("пользователь не найден")
	}
	f.user.PasswordHash = passwordHash
	f.user.PasswordChangeRequired = false
	return nil
}

type fakeAuthSettings struct{}

func (fakeAuthSettings) Get(string) (*models.SystemSetting, error) { return nil, nil }

type fakeAuthSessions struct {
	session *models.ServerSession
	hash    []byte
}

func (f *fakeAuthSessions) Create(userID uuid.UUID, hash []byte, expiresAt time.Time) (*models.ServerSession, error) {
	f.hash = append([]byte(nil), hash...)
	f.session = &models.ServerSession{ID: uuid.New(), UserID: userID, TokenHash: f.hash, ExpiresAt: expiresAt}
	return f.session, nil
}
func (f *fakeAuthSessions) GetActiveByTokenHash(hash []byte, _ time.Time) (*models.ServerSession, error) {
	if bytes.Equal(hash, f.hash) {
		return f.session, nil
	}
	return nil, nil
}
func (f *fakeAuthSessions) RevokeByTokenHash([]byte, time.Time) error { return nil }

func TestAuthLoginCreatesHashedSessionAndBearerAuthenticatesMe(t *testing.T) {
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	user := &models.User{ID: uuid.New(), Login: "admin", PasswordHash: hash, FullName: "Admin", IsActive: true}
	sessions := &fakeAuthSessions{}
	api := &managementAPI{
		cfg:          &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers:    &fakeAuthUsers{user: user},
		authSettings: fakeAuthSettings{},
		sessions:     sessions,
		audit:        &fakeAdminAudit{},
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"login":"admin","password":"Passw0rd!"}`))
	loginResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResponse, loginRequest)
	require.Equal(t, http.StatusOK, loginResponse.Code)
	var response struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResponse.Body).Decode(&response))
	require.NotEmpty(t, response.AccessToken)
	assert.Len(t, sessions.hash, 32)
	assert.NotEqual(t, []byte(response.AccessToken), sessions.hash)

	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+response.AccessToken)
	meResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(meResponse, meRequest)

	require.Equal(t, http.StatusOK, meResponse.Code)
	assert.Contains(t, meResponse.Body.String(), `"login":"admin"`)
}

func TestAuthMeRejectsMissingBearerToken(t *testing.T) {
	api := &managementAPI{sessions: &fakeAuthSessions{}}
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestChangeRequiredPasswordUpdatesCredentials(t *testing.T) {
	oldHash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	user := &models.User{ID: uuid.New(), Login: "user", PasswordHash: oldHash, FullName: "User", IsActive: true, PasswordChangeRequired: true}
	api := &managementAPI{
		authUsers:    &fakeAuthUsers{user: user},
		authSettings: fakeAuthSettings{},
		audit:        &fakeAdminAudit{},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-required-password", bytes.NewBufferString(`{"login":"user","oldPassword":"Passw0rd!","newPassword":"NewPassw0rd!"}`))
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	assert.True(t, security.VerifyPassword(user.PasswordHash, "NewPassw0rd!"))
	assert.False(t, user.PasswordChangeRequired)
}

func TestChangePasswordUsesBearerSessionAndInvalidatesOldCredentials(t *testing.T) {
	oldHash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	user := &models.User{ID: uuid.New(), Login: "user", PasswordHash: oldHash, FullName: "User", IsActive: true}
	sessions := &fakeAuthSessions{}
	api := &managementAPI{
		cfg:          &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers:    &fakeAuthUsers{user: user},
		authSettings: fakeAuthSettings{},
		sessions:     sessions,
		audit:        &fakeAdminAudit{},
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"login":"user","password":"Passw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code)
	var loginResponse struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&loginResponse))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBufferString(`{"oldPassword":"Passw0rd!","newPassword":"NewPassw0rd!"}`))
	request.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	assert.False(t, security.VerifyPassword(user.PasswordHash, "Passw0rd!"))
	assert.True(t, security.VerifyPassword(user.PasswordHash, "NewPassw0rd!"))
}

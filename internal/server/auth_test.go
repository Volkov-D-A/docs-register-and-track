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

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

type fakeAuthUsers struct{ user *models.User }

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
func (f *fakeAuthUsers) IncrementFailedLoginAttempts(uuid.UUID) (int, bool, error) {
	return 1, true, nil
}
func (f *fakeAuthUsers) ResetFailedLoginAttempts(uuid.UUID) error { return nil }

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

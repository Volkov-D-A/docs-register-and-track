package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

type parallelAuthUsers struct{ users map[uuid.UUID]*models.User }

func (s parallelAuthUsers) GetByLogin(login string) (*models.User, error) {
	for _, user := range s.users {
		if user.Login == login {
			return user, nil
		}
	}
	return nil, nil
}
func (s parallelAuthUsers) GetByID(id uuid.UUID) (*models.User, error) { return s.users[id], nil }
func (parallelAuthUsers) UpdatePassword(uuid.UUID, string) error       { return nil }
func (parallelAuthUsers) IncrementFailedLoginAttempts(uuid.UUID) (int, bool, error) {
	return 0, true, nil
}
func (parallelAuthUsers) ResetFailedLoginAttempts(uuid.UUID) error { return nil }

type parallelSessions struct {
	sessions map[string]*models.ServerSession
}

func (parallelSessions) Create(uuid.UUID, []byte, time.Time) (*models.ServerSession, error) {
	return nil, nil
}
func (s parallelSessions) GetActiveByTokenHash(hash []byte, _ time.Time) (*models.ServerSession, error) {
	return s.sessions[string(hash)], nil
}
func (parallelSessions) RevokeByTokenHash([]byte, time.Time) error { return nil }

type fakeUserManagementStore struct {
	users         []models.User
	created       models.CreateUserRequest
	updated       models.UpdateUserRequest
	resetID       uuid.UUID
	resetPassword string
	effects       []models.OutboxEvent
}

func (s *fakeUserManagementStore) GetAll() ([]models.User, error) { return s.users, nil }
func (s *fakeUserManagementStore) GetByID(id uuid.UUID) (*models.User, error) {
	for i := range s.users {
		if s.users[i].ID == id {
			return &s.users[i], nil
		}
	}
	return nil, nil
}
func (s *fakeUserManagementStore) CreateWithOutbox(req models.CreateUserRequest, effects []models.OutboxEvent) (*models.User, error) {
	s.created, s.effects = req, effects
	user := models.User{ID: uuid.New(), Login: req.Login, FullName: req.FullName, IsActive: true}
	s.users = append(s.users, user)
	return &user, nil
}
func (s *fakeUserManagementStore) UpdateWithOutbox(req models.UpdateUserRequest, effects []models.OutboxEvent) (*models.User, error) {
	s.updated, s.effects = req, effects
	id, _ := uuid.Parse(req.ID)
	return &models.User{ID: id, Login: req.Login, FullName: req.FullName, IsActive: req.IsActive}, nil
}
func (s *fakeUserManagementStore) ResetPasswordWithOutbox(id uuid.UUID, password string, effects []models.OutboxEvent) error {
	s.resetID, s.resetPassword, s.effects = id, password, effects
	return nil
}

func authenticatedUserAPI(t *testing.T, permissions []string) (*managementAPI, *fakeUserManagementStore, string) {
	t.Helper()
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	admin := &models.User{ID: uuid.New(), Login: "admin", PasswordHash: hash, FullName: "Admin", IsActive: true, SystemPermissions: permissions}
	sessions := &fakeAuthSessions{}
	store := &fakeUserManagementStore{users: []models.User{{ID: uuid.New(), Login: "target", FullName: "Target", IsActive: true}}}
	api := &managementAPI{
		cfg:       &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers: &fakeAuthUsers{user: admin}, authSettings: fakeAuthSettings{}, sessions: sessions,
		audit: &fakeAdminAudit{}, userCommands: store,
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"admin","password":"Passw0rd!"}`))
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, login)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var body struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	return api, store, body.AccessToken
}

func TestUserAPIRequiresAdminPermission(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, req)

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Contains(t, response.Body.String(), `"code":"forbidden"`)
}

func TestUserAPIKeepsParallelRequestPrincipalsIsolated(t *testing.T) {
	admin := &models.User{ID: uuid.New(), Login: "admin", IsActive: true, SystemPermissions: []string{models.SystemPermissionAdmin}}
	regular := &models.User{ID: uuid.New(), Login: "regular", IsActive: true}
	adminHash := sha256.Sum256([]byte("admin-token"))
	regularHash := sha256.Sum256([]byte("regular-token"))
	api := &managementAPI{
		authUsers: parallelAuthUsers{users: map[uuid.UUID]*models.User{admin.ID: admin, regular.ID: regular}},
		sessions: parallelSessions{sessions: map[string]*models.ServerSession{
			string(adminHash[:]):   {ID: uuid.New(), UserID: admin.ID, ExpiresAt: time.Now().Add(time.Hour)},
			string(regularHash[:]): {ID: uuid.New(), UserID: regular.ID, ExpiresAt: time.Now().Add(time.Hour)},
		}},
		userCommands: &fakeUserManagementStore{},
	}
	type result struct {
		admin  bool
		status int
	}
	results := make(chan result, 80)
	for i := 0; i < 40; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
			req.Header.Set("Authorization", "Bearer admin-token")
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, req)
			results <- result{admin: true, status: response.Code}
		}()
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
			req.Header.Set("Authorization", "Bearer regular-token")
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, req)
			results <- result{status: response.Code}
		}()
	}
	for i := 0; i < 80; i++ {
		got := <-results
		if got.admin {
			assert.Equal(t, http.StatusOK, got.status)
		} else {
			assert.Equal(t, http.StatusForbidden, got.status)
		}
	}
}

func TestUserAPICreateGeneratesTemporaryPasswordWithoutLeakingItToOutbox(t *testing.T) {
	api, store, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"login":"new-user","fullName":"New User","isDocumentParticipant":true}`))
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, req)

	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	assert.NotEmpty(t, store.created.Password)
	assert.True(t, store.created.PasswordChangeRequired)
	require.Len(t, store.effects, 1)
	assert.NotContains(t, store.effects[0].Payload, store.created.Password)
	assert.NotContains(t, store.effects[0].Payload, "temporaryPassword")
	var result map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
	assert.Equal(t, store.created.Password, result["temporaryPassword"])
}

func TestUserAPIUpdateUsesPathID(t *testing.T) {
	api, store, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	id := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+id.String(), bytes.NewBufferString(`{"id":"`+uuid.NewString()+`","login":"updated","fullName":"Updated","isActive":true}`))
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, req)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, id.String(), store.updated.ID)
}

func TestUserAPIResetReturnsPasswordOnlyInSuccessfulResponse(t *testing.T) {
	api, store, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	id := store.users[0].ID
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+id.String()+"/reset-password", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, req)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, id, store.resetID)
	assert.NoError(t, security.ValidatePassword(store.resetPassword))
	require.Len(t, store.effects, 1)
	assert.NotContains(t, store.effects[0].Payload, store.resetPassword)
	assert.Contains(t, response.Body.String(), store.resetPassword)
}

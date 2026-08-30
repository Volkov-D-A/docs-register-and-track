package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestServerAuthSessionLifecycleIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	passwordHash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	userID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'session-integration', $2, 'Session Integration', TRUE, FALSE)
	`, userID, passwordHash)
	require.NoError(t, err)

	cfg := validConfig()
	api := newManagementAPI(&App{db: db, cfg: cfg})
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"login":"session-integration","password":"Passw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
	var response struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&response))
	require.NotEmpty(t, response.AccessToken)

	var activeSessions int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM server_sessions WHERE user_id = $1 AND revoked_at IS NULL`, userID).Scan(&activeSessions))
	assert.Equal(t, 1, activeSessions)

	logout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logout.Header.Set("Authorization", "Bearer "+response.AccessToken)
	logoutResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(logoutResult, logout)
	require.Equal(t, http.StatusNoContent, logoutResult.Code)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM server_sessions WHERE user_id = $1 AND revoked_at IS NULL`, userID).Scan(&activeSessions))
	assert.Zero(t, activeSessions)
}

func TestServerPasswordChangeRevokesAllSessionsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	passwordHash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	userID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'password-change-integration', $2, 'Password Change Integration', TRUE, FALSE)
	`, userID, passwordHash)
	require.NoError(t, err)

	api := newManagementAPI(&App{db: db, cfg: validConfig()})
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"login":"password-change-integration","password":"Passw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
	var response struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&response))

	change := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBufferString(`{"oldPassword":"Passw0rd!","newPassword":"NewPassw0rd!"}`))
	change.Header.Set("Authorization", "Bearer "+response.AccessToken)
	changeResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(changeResult, change)
	require.Equal(t, http.StatusNoContent, changeResult.Code, changeResult.Body.String())

	var activeSessions int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM server_sessions WHERE user_id = $1 AND revoked_at IS NULL`, userID).Scan(&activeSessions))
	assert.Zero(t, activeSessions)
	me := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	me.Header.Set("Authorization", "Bearer "+response.AccessToken)
	meResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(meResult, me)
	assert.Equal(t, http.StatusUnauthorized, meResult.Code)
}

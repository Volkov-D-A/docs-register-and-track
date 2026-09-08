package server

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	api := newIntegrationManagementAPI(t, &App{db: db, cfg: cfg})
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

	api := newIntegrationManagementAPI(t, &App{db: db, cfg: validConfig()})
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

// Repeated locks must each produce an audit entry through the production HTTP path.
func TestServerRepeatedLockoutsAuditIntegration(t *testing.T) {
	db := &database.DB{DB: integrationdb.Open(t)}
	hash, err := security.HashPassword("Passw0rd!")
	require.NoError(t, err)
	id := uuid.New()
	_, err = db.Exec(`INSERT INTO users(id,login,password_hash,full_name,is_active,password_change_required) VALUES($1,'lock-audit',$2,'Lock Audit',TRUE,FALSE)`, id, hash)
	require.NoError(t, err)
	adminID := uuid.New()
	_, err = db.Exec(`INSERT INTO users(id,login,password_hash,full_name,is_active,password_change_required) VALUES($1,'lock-audit-admin',$2,'Admin',TRUE,FALSE)`, adminID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions(user_id,permission,is_allowed) VALUES($1,'admin',TRUE)`, adminID)
	require.NoError(t, err)
	api := newIntegrationManagementAPI(t, &App{db: db, cfg: validConfig()})
	for cycle := 0; cycle < 2; cycle++ {
		if cycle > 0 {
			// Fixture reactivation separates two independent account lock transitions.
			_, err = db.Exec(`UPDATE users SET is_active=TRUE,failed_login_attempts=0 WHERE id=$1`, id)
			require.NoError(t, err)
		}
		for attempt := 1; attempt <= 5; attempt++ {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"login":"lock-audit","password":"wrong"}`))
			// Exercise the account counter independently of per-address throttling.
			request.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", cycle*5+attempt)
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, request)
			status := http.StatusUnauthorized
			if attempt == 5 {
				status = http.StatusForbidden
			}
			require.Equal(t, status, response.Code, response.Body.String())
		}
		var active bool
		var attempts, count int
		require.NoError(t, db.QueryRow(`SELECT is_active,failed_login_attempts FROM users WHERE id=$1`, id).Scan(&active, &attempts))
		require.False(t, active)
		require.Equal(t, 5, attempts)
		require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT id) FROM admin_audit_log WHERE user_id=$1 AND action='USER_LOCKED'`, id).Scan(&count))
		require.Equal(t, cycle+1, count)
	}
}

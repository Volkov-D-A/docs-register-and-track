package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestUserAPIPersistsChangeOutboxAndSessionRevocationIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	users := repository.NewUserRepository(db)
	users.SetOutbox(repository.NewOutboxRepository(db))
	sessions := repository.NewServerSessionRepository(db)
	hash, err := security.HashPassword("AdminPassw0rd!")
	require.NoError(t, err)
	adminID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active) VALUES ($1, 'api-admin', $2, 'API Admin', TRUE)`, adminID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, $2, TRUE)`, adminID, models.SystemPermissionAdmin)
	require.NoError(t, err)

	api := &managementAPI{
		cfg:       &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers: users, authSettings: repository.NewSettingsRepository(db), sessions: sessions,
		audit: repository.NewAdminAuditLogRepository(db), userCommands: users,
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"api-admin","password":"AdminPassw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
	var loginBody struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&loginBody))

	create := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"login":"api-target","fullName":"API Target","isDocumentParticipant":true}`))
	create.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	createResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResult, create)
	require.Equal(t, http.StatusCreated, createResult.Code, createResult.Body.String())
	var created struct{ ID, TemporaryPassword string }
	require.NoError(t, json.NewDecoder(createResult.Body).Decode(&created))
	require.NotEmpty(t, created.TemporaryPassword)
	targetID := uuid.MustParse(created.ID)
	_, err = sessions.Create(targetID, []byte("target-session-hash-32-bytes-long!"), time.Now().Add(time.Hour))
	require.NoError(t, err)

	update := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+created.ID, strings.NewReader(`{"login":"api-target","fullName":"API Target","isActive":false,"isDocumentParticipant":true}`))
	update.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	updateResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(updateResult, update)
	require.Equal(t, http.StatusOK, updateResult.Code, updateResult.Body.String())

	var revokedSessions, pendingEffects int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM server_sessions WHERE user_id=$1 AND revoked_at IS NOT NULL`, targetID).Scan(&revokedSessions))
	require.Equal(t, 1, revokedSessions)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE processed_at IS NULL AND event_type=$1`, models.OutboxEventAudit).Scan(&pendingEffects))
	require.Equal(t, 2, pendingEffects)
}

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestSettingsAPIPersistsUpdatesWithAuditOutboxIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	outbox := repository.NewOutboxRepository(db)
	users := repository.NewUserRepository(db)
	users.SetOutbox(outbox)
	settings := repository.NewSettingsRepository(db)
	settings.SetOutbox(outbox)
	sessions := repository.NewServerSessionRepository(db)

	hash, err := security.HashPassword("SettingsPassw0rd!")
	require.NoError(t, err)
	userID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'settings-admin', $2, 'Settings Admin', TRUE, FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, $2, TRUE)`, userID, models.SystemPermissionAdmin)
	require.NoError(t, err)

	api := &managementAPI{
		cfg:       &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers: users, authSettings: settings, sessions: sessions, settings: settings,
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"settings-admin","password":"SettingsPassw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
	var loginBody struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&loginBody))

	current, err := settings.Get("max_file_size_mb")
	require.NoError(t, err)
	firstValue := "16"
	if current.Value == firstValue {
		firstValue = "17"
	}
	for _, value := range []string{firstValue, current.Value} {
		request := httptest.NewRequest(http.MethodPatch, "/api/v1/settings/max_file_size_mb", strings.NewReader(`{"value":"`+value+`"}`))
		request.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	}

	updated, err := settings.Get("max_file_size_mb")
	require.NoError(t, err)
	require.Equal(t, current.Value, updated.Value)
	var effects, distinctKeys int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT deduplication_key) FROM event_outbox WHERE event_type=$1 AND deduplication_key LIKE 'setting:max_file_size_mb:update:%'`, models.OutboxEventAudit).Scan(&effects, &distinctKeys))
	require.Equal(t, 2, effects)
	require.Equal(t, 2, distinctKeys)
}

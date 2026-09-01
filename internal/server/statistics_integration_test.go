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
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestDashboardAndStatisticsAPIEnforceServerPermissionsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	password := "StatisticsPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	userID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'statistics-reader', $2, 'Statistics Reader', TRUE, FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, 'admin', TRUE)`, userID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, $2, TRUE)`, userID, models.SystemPermissionStatsDocuments)
	require.NoError(t, err)

	api := newManagementAPI(&App{db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, metrics: observability.NewRegistry(32)})
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"statistics-reader","password":"`+password+`"}`))
	loginResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResponse, login)
	require.Equal(t, http.StatusOK, loginResponse.Code, loginResponse.Body.String())
	var session struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResponse.Body).Decode(&session))
	request := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+session.AccessToken)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, req)
		return response
	}

	dashboard := request("/api/v1/dashboard/activity")
	require.Equal(t, http.StatusOK, dashboard.Code, dashboard.Body.String())
	documents := request("/api/v1/statistics/documents")
	require.Equal(t, http.StatusOK, documents.Code, documents.Body.String())
	require.Contains(t, documents.Body.String(), `"totalYear":0`)
	system := request("/api/v1/statistics/system")
	require.Equal(t, http.StatusForbidden, system.Code, system.Body.String())
}

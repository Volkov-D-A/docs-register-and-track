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

func TestReferenceAPIPersistsMutationsWithAuditOutboxIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	outbox := repository.NewOutboxRepository(db)
	users := repository.NewUserRepository(db)
	users.SetOutbox(outbox)
	references := repository.NewReferenceRepository(db)
	references.SetOutbox(outbox)
	sessions := repository.NewServerSessionRepository(db)
	hash, err := security.HashPassword("ReferencePassw0rd!")
	require.NoError(t, err)
	userID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'reference-manager', $2, 'Reference Manager', TRUE, FALSE)`, userID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES
		($1, $2, TRUE), ($1, $3, TRUE)`, userID, models.SystemPermissionAdmin, models.SystemPermissionReferences)
	require.NoError(t, err)

	api := &managementAPI{
		cfg:       &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers: users, authSettings: repository.NewSettingsRepository(db), sessions: sessions,
		references: references,
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"reference-manager","password":"ReferencePassw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
	var loginBody struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&loginBody))
	token := loginBody.AccessToken

	resolve := func(path, name string) string {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"`+name+`"}`))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var item struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&item))
		return item.ID
	}
	requestNoContent := func(method, path, body string) {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	}

	organizationID := resolve("/api/v1/references/organizations/resolve", "Source Organization")
	targetID := resolve("/api/v1/references/organizations/resolve", "Target Organization")
	requestNoContent(http.MethodPatch, "/api/v1/references/organizations/"+organizationID, `{"name":"Updated Organization"}`)
	requestNoContent(http.MethodPost, "/api/v1/references/organizations/"+organizationID+"/merge", `{"targetId":"`+targetID+`"}`)
	requestNoContent(http.MethodDelete, "/api/v1/references/organizations/"+targetID, "")

	executorID := resolve("/api/v1/references/resolution-executors/resolve", "Executor")
	requestNoContent(http.MethodPatch, "/api/v1/references/resolution-executors/"+executorID, `{"name":"Chief Executor"}`)
	requestNoContent(http.MethodDelete, "/api/v1/references/resolution-executors/"+executorID, "")

	var organizations, executors, effects int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM organizations`).Scan(&organizations))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM resolution_executors`).Scan(&executors))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE processed_at IS NULL AND event_type=$1`, models.OutboxEventAudit).Scan(&effects))
	require.Zero(t, organizations)
	require.Zero(t, executors)
	require.Equal(t, 5, effects)
}

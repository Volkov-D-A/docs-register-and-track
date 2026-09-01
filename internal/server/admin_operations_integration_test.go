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
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestAdminOperationsAPIPersistsRequeueIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	password := "AdminOperationsPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	adminID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'operations-admin', $2, 'Operations Admin', TRUE, FALSE)`, adminID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, 'admin', TRUE)`, adminID)
	require.NoError(t, err)
	auditID, err := repository.NewAdminAuditLogRepository(db).Create(models.CreateAdminAuditLogRequest{UserID: adminID, UserName: "Operations Admin", Action: "INTEGRATION_AUDIT", Details: "server-owned"})
	require.NoError(t, err)
	outbox := repository.NewOutboxRepository(db)
	event := models.OutboxEvent{EventType: models.OutboxEventAudit, DeduplicationKey: "admin-operations:" + uuid.NewString(), Payload: `{"test":true}`}
	require.NoError(t, outbox.Enqueue(event))
	var eventID uuid.UUID
	require.NoError(t, db.QueryRow(`SELECT id FROM event_outbox WHERE deduplication_key=$1`, event.DeduplicationKey).Scan(&eventID))
	require.NoError(t, outbox.MarkFailed(eventID, 5, time.Second, 5, "terminal integration failure"))

	api := newManagementAPI(&App{db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, metrics: observability.NewRegistry(32)})
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"operations-admin","password":"`+password+`"}`))
	loginResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResponse, login)
	require.Equal(t, http.StatusOK, loginResponse.Code, loginResponse.Body.String())
	var session struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResponse.Body).Decode(&session))
	request := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+session.AccessToken)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, req)
		return response
	}

	audit := request(http.MethodGet, "/api/v1/admin/audit?page=1&pageSize=20")
	require.Equal(t, http.StatusOK, audit.Code, audit.Body.String())
	require.Contains(t, audit.Body.String(), auditID.String())
	failed := request(http.MethodGet, "/api/v1/admin/outbox/failed?limit=100")
	require.Equal(t, http.StatusOK, failed.Code, failed.Body.String())
	require.Contains(t, failed.Body.String(), eventID.String())
	requeue := request(http.MethodPost, "/api/v1/admin/outbox/"+eventID.String()+"/requeue")
	require.Equal(t, http.StatusNoContent, requeue.Code, requeue.Body.String())
	var failedAt *time.Time
	require.NoError(t, db.QueryRow(`SELECT failed_at FROM event_outbox WHERE id=$1`, eventID).Scan(&failedAt))
	require.Nil(t, failedAt)
}

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

func TestAssignmentAPIUsesServerPrincipalAndPersistsCoExecutorsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	password := "AssignmentPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	managerID, executorID, coExecutorID := uuid.New(), uuid.New(), uuid.New()
	for _, user := range []struct {
		id    uuid.UUID
		login string
		name  string
	}{{managerID, "assignment-manager", "Assignment Manager"}, {executorID, "assignment-executor", "Assignment Executor"}, {coExecutorID, "assignment-coexecutor", "Assignment Coexecutor"}} {
		_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, is_document_participant, password_change_required)
			VALUES ($1, $2, $3, $4, TRUE, TRUE, FALSE)`, user.id, user.login, hash, user.name)
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
		VALUES ('outgoing_letter', 'user', $1, 'read', TRUE),
		       ('outgoing_letter', 'user', $1, 'assign', TRUE)`, managerID.String())
	require.NoError(t, err)
	nomenclatureID, organizationID := uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
		VALUES ($1, 'Outgoing assignments', 'OUT-A', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomenclatureID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations (id, name) VALUES ($1, 'Assignment Organization')`, organizationID)
	require.NoError(t, err)
	document, err := repository.NewOutgoingDocumentRepository(db).Create(models.CreateOutgoingDocRequest{
		NomenclatureID: nomenclatureID, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter,
		RecipientOrgID: organizationID, CreatedBy: managerID, OutgoingDate: time.Now().UTC(), Content: "assignment api integration",
		PagesCount: 1, SenderSignatory: "Signer", SenderExecutor: "Executor", Addressee: "Addressee",
	})
	require.NoError(t, err)

	api := newManagementAPI(&App{db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, metrics: observability.NewRegistry(32)})
	login := func(login string) string {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"`+login+`","password":"`+password+`"}`))
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var session struct {
			AccessToken string `json:"accessToken"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&session))
		return session.AccessToken
	}

	managerToken := login("assignment-manager")
	createBody := `{"documentId":"` + document.ID.String() + `","executorId":"` + executorID.String() + `","content":"Server-owned assignment","deadline":"2026-09-30","coExecutorIds":["` + coExecutorID.String() + `"]}`
	create := httptest.NewRequest(http.MethodPost, "/api/v1/assignments", strings.NewReader(createBody))
	create.Header.Set("Authorization", "Bearer "+managerToken)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, create)
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(createResponse.Body).Decode(&created))
	require.NotEmpty(t, created.ID)

	var coExecutorCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM assignment_co_executors WHERE assignment_id = $1 AND user_id = $2`, created.ID, coExecutorID).Scan(&coExecutorCount))
	require.Equal(t, 1, coExecutorCount)

	executorToken := login("assignment-executor")
	status := httptest.NewRequest(http.MethodPatch, "/api/v1/assignments/"+created.ID+"/status", strings.NewReader(`{"status":"in_progress","report":""}`))
	status.Header.Set("Authorization", "Bearer "+executorToken)
	statusResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(statusResponse, status)
	require.Equal(t, http.StatusOK, statusResponse.Code, statusResponse.Body.String())
	require.Contains(t, statusResponse.Body.String(), `"status":"in_progress"`)
}

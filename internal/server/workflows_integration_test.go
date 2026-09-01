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

func TestWorkflowAPIPersistsAcknowledgmentAndScopesUserEventsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	password := "WorkflowPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	managerID, recipientID, outsiderID := uuid.New(), uuid.New(), uuid.New()
	for _, user := range []struct {
		id    uuid.UUID
		login string
		name  string
	}{{managerID, "workflow-manager", "Workflow Manager"}, {recipientID, "workflow-recipient", "Workflow Recipient"}, {outsiderID, "workflow-outsider", "Workflow Outsider"}} {
		_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, is_document_participant, password_change_required)
			VALUES ($1, $2, $3, $4, TRUE, TRUE, FALSE)`, user.id, user.login, hash, user.name)
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
		VALUES ('outgoing_letter', 'user', $1, 'read', TRUE),
		       ('outgoing_letter', 'user', $1, 'acknowledge', TRUE)`, managerID.String())
	require.NoError(t, err)
	nomenclatureID, organizationID := uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
		VALUES ($1, 'Workflow acknowledgments', 'OUT-W', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomenclatureID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations (id, name) VALUES ($1, 'Workflow Organization')`, organizationID)
	require.NoError(t, err)
	document, err := repository.NewOutgoingDocumentRepository(db).Create(models.CreateOutgoingDocRequest{
		NomenclatureID: nomenclatureID, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter,
		RecipientOrgID: organizationID, CreatedBy: managerID, OutgoingDate: time.Now().UTC(), Content: "workflow api integration",
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

	managerToken := login("workflow-manager")
	createBody := `{"documentId":"` + document.ID.String() + `","content":"Read the document","userIds":["` + recipientID.String() + `"]}`
	create := httptest.NewRequest(http.MethodPost, "/api/v1/acknowledgments", strings.NewReader(createBody))
	create.Header.Set("Authorization", "Bearer "+managerToken)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, create)
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	var acknowledgment dtoAcknowledgmentID
	require.NoError(t, json.NewDecoder(createResponse.Body).Decode(&acknowledgment))

	recipientToken := login("workflow-recipient")
	pending := httptest.NewRequest(http.MethodGet, "/api/v1/acknowledgments/pending", nil)
	pending.Header.Set("Authorization", "Bearer "+recipientToken)
	pendingResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(pendingResponse, pending)
	require.Equal(t, http.StatusOK, pendingResponse.Code, pendingResponse.Body.String())
	require.Contains(t, pendingResponse.Body.String(), acknowledgment.ID)

	confirm := httptest.NewRequest(http.MethodPost, "/api/v1/acknowledgments/"+acknowledgment.ID+"/confirm", nil)
	confirm.Header.Set("Authorization", "Bearer "+recipientToken)
	confirmResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(confirmResponse, confirm)
	require.Equal(t, http.StatusNoContent, confirmResponse.Code, confirmResponse.Body.String())
	var confirmed bool
	require.NoError(t, db.QueryRow(`SELECT confirmed_at IS NOT NULL FROM acknowledgment_users WHERE acknowledgment_id=$1 AND user_id=$2`, acknowledgment.ID, recipientID).Scan(&confirmed))
	require.True(t, confirmed)

	event, err := repository.NewUserEventRepository(db).Create(models.CreateUserEventRequest{
		RecipientUserID: recipientID, ActorUserID: &managerID, DocumentID: document.ID,
		DocumentKind: string(models.DocumentKindOutgoingLetter), EntityType: models.UserEventEntityAcknowledgment,
		EntityID: uuid.MustParse(acknowledgment.ID), EventType: models.UserEventAcknowledgmentCreated, Title: "Scoped event", Message: "Recipient only",
	})
	require.NoError(t, err)
	queryEvents := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/user-events/query", strings.NewReader(`{"page":1,"pageSize":20}`))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		return response
	}
	recipientEvents := queryEvents(recipientToken)
	require.Equal(t, http.StatusOK, recipientEvents.Code, recipientEvents.Body.String())
	require.Contains(t, recipientEvents.Body.String(), event.ID.String())
	outsiderEvents := queryEvents(login("workflow-outsider"))
	require.Equal(t, http.StatusOK, outsiderEvents.Code, outsiderEvents.Body.String())
	require.NotContains(t, outsiderEvents.Body.String(), event.ID.String())
}

type dtoAcknowledgmentID struct {
	ID string `json:"id"`
}

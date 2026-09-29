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

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestWorkflowAPIPersistsAcknowledgmentAndScopesUserEventsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	password := "WorkflowPassw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)
	managerID, otherManagerID, recipientID, outsiderID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, user := range []struct {
		id    uuid.UUID
		login string
		name  string
	}{{managerID, "workflow-manager", "Workflow Manager"}, {otherManagerID, "workflow-other-manager", "Other Manager"}, {recipientID, "workflow-recipient", "Workflow Recipient"}, {outsiderID, "workflow-outsider", "Workflow Outsider"}} {
		_, err = db.Exec(`INSERT INTO users (id, login, password_hash, last_name, first_name, no_patronymic, is_active, is_document_participant, password_change_required)
			VALUES ($1, $2, $3, $4, 'User', TRUE, TRUE, TRUE, FALSE)`, user.id, user.login, hash, user.name)
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
		VALUES ('outgoing_letter', 'user', $1, 'read', TRUE),
		       ('outgoing_letter', 'user', $1, 'assign', TRUE),
		       ('outgoing_letter', 'user', $2, 'read', TRUE),
		       ('outgoing_letter', 'user', $2, 'assign', TRUE)`, managerID.String(), otherManagerID.String())
	require.NoError(t, err)
	nomenclatureID, organizationID := uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
		VALUES ($1, 'Workflow acknowledgments', 'OUT-W', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomenclatureID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations (id, name) VALUES ($1, 'Workflow Organization')`, organizationID)
	require.NoError(t, err)
	documentRepo := repository.NewOutgoingDocumentRepository(db)
	documentRepo.SetOutbox(repository.NewOutboxRepository(db))
	document, err := documentRepo.CreateWithJournal(models.CreateOutgoingDocRequest{
		NomenclatureID: nomenclatureID, IdempotencyKey: uuid.New(), DocumentTypeID: models.DocumentTypeLetter,
		RecipientOrgID: organizationID, CreatedBy: managerID, OutgoingDate: time.Now().UTC(), Content: "workflow api integration",
		PagesCount: 1, SenderSignatory: "Signer", SenderExecutor: "Executor", Addressee: "Addressee",
	}, "CREATE", "Created %s")
	require.NoError(t, err)

	api := newIntegrationManagementAPI(t, &App{db: db, cfg: &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}}, metrics: observability.NewRegistry(32)})
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

	removedRoute := httptest.NewRecorder()
	api.Handler().ServeHTTP(removedRoute, httptest.NewRequest(http.MethodPost, "/api/v1/acknowledgments", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusNotFound, removedRoute.Code)
	managerToken := login("workflow-manager")
	inactiveID, nonParticipantID := uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, last_name, first_name, no_patronymic, is_active, is_document_participant)
		VALUES ($1, 'workflow-inactive', $3, 'Inactive', 'Recipient', TRUE, FALSE, TRUE),
		       ($2, 'workflow-non-participant', $3, 'Nonparticipant', 'Recipient', TRUE, TRUE, FALSE)`, inactiveID, nonParticipantID, hash)
	require.NoError(t, err)
	for _, recipientID := range []uuid.UUID{inactiveID, nonParticipantID, uuid.New()} {
		body := `{"type":"acknowledgment","documentId":"` + document.ID.String() + `","userIds":["` + recipientID.String() + `"]}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/assignments", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+managerToken)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	}
	invalidBody := `{"type":"acknowledgment","documentId":"` + document.ID.String() + `","userIds":["` + recipientID.String() + `","not-a-uuid"]}`
	invalidCreate := httptest.NewRequest(http.MethodPost, "/api/v1/assignments", strings.NewReader(invalidBody))
	invalidCreate.Header.Set("Authorization", "Bearer "+managerToken)
	invalidResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(invalidResponse, invalidCreate)
	require.Equal(t, http.StatusBadRequest, invalidResponse.Code, invalidResponse.Body.String())
	var acknowledgmentCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM assignments WHERE document_id = $1`, document.ID).Scan(&acknowledgmentCount))
	require.Zero(t, acknowledgmentCount)

	duplicateBody := `{"type":"acknowledgment","documentId":"` + document.ID.String() + `","userIds":["` + recipientID.String() + `","` + recipientID.String() + `"]}`
	duplicateCreate := httptest.NewRequest(http.MethodPost, "/api/v1/assignments", strings.NewReader(duplicateBody))
	duplicateCreate.Header.Set("Authorization", "Bearer "+managerToken)
	duplicateResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(duplicateResponse, duplicateCreate)
	require.Equal(t, http.StatusCreated, duplicateResponse.Code, duplicateResponse.Body.String())
	var duplicateAcknowledgment dtoAssignmentID
	require.NoError(t, json.NewDecoder(duplicateResponse.Body).Decode(&duplicateAcknowledgment))
	var recipientCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM assignment_recipients WHERE assignment_id = $1`, duplicateAcknowledgment.ID).Scan(&recipientCount))
	require.Equal(t, 1, recipientCount)

	createBody := `{"type":"acknowledgment","documentId":"` + document.ID.String() + `","userIds":["` + recipientID.String() + `"]}`
	create := httptest.NewRequest(http.MethodPost, "/api/v1/assignments", strings.NewReader(createBody))
	create.Header.Set("Authorization", "Bearer "+managerToken)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, create)
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	var acknowledgment dtoAssignmentID
	require.NoError(t, json.NewDecoder(createResponse.Body).Decode(&acknowledgment))

	recipientToken := login("workflow-recipient")
	pending := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/query", strings.NewReader(`{"types":["acknowledgment"],"mode":"execution","page":1,"pageSize":100}`))
	pending.Header.Set("Authorization", "Bearer "+recipientToken)
	pendingResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(pendingResponse, pending)
	require.Equal(t, http.StatusOK, pendingResponse.Code, pendingResponse.Body.String())
	require.Contains(t, pendingResponse.Body.String(), acknowledgment.ID)

	confirm := httptest.NewRequest(http.MethodPatch, "/api/v1/assignments/"+acknowledgment.ID+"/status", strings.NewReader(`{"status":"finished"}`))
	confirm.Header.Set("Authorization", "Bearer "+recipientToken)
	confirmResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(confirmResponse, confirm)
	require.Equal(t, http.StatusOK, confirmResponse.Code, confirmResponse.Body.String())
	var confirmed bool
	require.NoError(t, db.QueryRow(`SELECT confirmed_at IS NOT NULL FROM assignment_recipients WHERE assignment_id=$1 AND user_id=$2`, acknowledgment.ID, recipientID).Scan(&confirmed))
	require.True(t, confirmed)

	// Acknowledgments use the common API and list, sharing assign permission.
	requestTask := func(method, path, body, token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		return response
	}
	_, err = db.Exec(`INSERT INTO document_permissions(kind_code,subject_type,subject_key,action,is_allowed) VALUES ('outgoing_letter','user',$1,'assign',TRUE),('outgoing_letter','user',$1,'read',TRUE)`, outsiderID.String())
	require.NoError(t, err)
	assignOnlyToken := login("workflow-outsider")
	taskBody := `{"type":"acknowledgment","documentId":"` + document.ID.String() + `","content":"Read together","deadline":"2026-12-31","userIds":["` + recipientID.String() + `","` + otherManagerID.String() + `"]}`
	require.Equal(t, http.StatusForbidden, requestTask(http.MethodPost, "/api/v1/assignments", taskBody, recipientToken).Code)
	typedResponse := requestTask(http.MethodPost, "/api/v1/assignments", taskBody, managerToken)
	require.Equal(t, http.StatusCreated, typedResponse.Code, typedResponse.Body.String())
	var typed dto.Assignment
	require.NoError(t, json.Unmarshal(typedResponse.Body.Bytes(), &typed))
	require.Equal(t, models.AssignmentTypeAcknowledgment, typed.Type)
	require.Empty(t, typed.ExecutorID)
	require.Len(t, typed.Users, 2)
	require.Equal(t, "2026-12-31", typed.Deadline.Format("2006-01-02"))
	require.Empty(t, typed.Content)
	updatedTask := requestTask(http.MethodPatch, "/api/v1/assignments/"+typed.ID, `{"content":"Legacy update comment","deadline":"2026-12-30"}`, managerToken)
	require.Equal(t, http.StatusOK, updatedTask.Code, updatedTask.Body.String())
	var updated dto.Assignment
	require.NoError(t, json.Unmarshal(updatedTask.Body.Bytes(), &updated))
	require.Empty(t, updated.Content)
	require.Equal(t, "2026-12-30", updated.Deadline.Format("2006-01-02"))
	var storedContent string
	require.NoError(t, db.QueryRow(`SELECT content FROM assignments WHERE id=$1`, typed.ID).Scan(&storedContent))
	require.Empty(t, storedContent)

	executionBody := `{"type":"execution","documentId":"` + document.ID.String() + `","executorId":"` + recipientID.String() + `","content":"Execute"}`
	require.Equal(t, http.StatusForbidden, requestTask(http.MethodPost, "/api/v1/assignments", executionBody, recipientToken).Code)
	executionResponse := requestTask(http.MethodPost, "/api/v1/assignments", executionBody, assignOnlyToken)
	require.Equal(t, http.StatusCreated, executionResponse.Code, executionResponse.Body.String())
	var execution dto.Assignment
	require.NoError(t, json.Unmarshal(executionResponse.Body.Bytes(), &execution))
	query := `{"mode":"control","page":1,"pageSize":100}`
	controlled := requestTask(http.MethodPost, "/api/v1/assignments/query", query, managerToken)
	require.Equal(t, http.StatusOK, controlled.Code, controlled.Body.String())
	require.Contains(t, controlled.Body.String(), typed.ID)
	require.Contains(t, controlled.Body.String(), execution.ID)
	controlled = requestTask(http.MethodPost, "/api/v1/assignments/query", query, assignOnlyToken)
	require.Equal(t, http.StatusOK, controlled.Code, controlled.Body.String())
	require.Contains(t, controlled.Body.String(), typed.ID)
	require.Contains(t, controlled.Body.String(), execution.ID)
	statusPath := "/api/v1/assignments/" + typed.ID + "/status"
	require.Equal(t, http.StatusForbidden, requestTask(http.MethodPatch, statusPath, `{"status":"finished"}`, managerToken).Code)
	require.Equal(t, http.StatusBadRequest, requestTask(http.MethodPatch, statusPath, `{"status":"completed","report":"Read"}`, recipientToken).Code)
	require.Equal(t, http.StatusForbidden, requestTask(http.MethodDelete, "/api/v1/assignments/"+typed.ID, "", assignOnlyToken).Code)
	firstConfirmed := requestTask(http.MethodPatch, statusPath, `{"status":"finished"}`, recipientToken)
	require.Equal(t, http.StatusOK, firstConfirmed.Code, firstConfirmed.Body.String())
	var afterFirst dto.Assignment
	require.NoError(t, json.Unmarshal(firstConfirmed.Body.Bytes(), &afterFirst))
	require.Equal(t, "new", afterFirst.Status)
	require.False(t, afterFirst.CanAct)
	personal := requestTask(http.MethodPost, "/api/v1/assignments/query", `{"types":[],"mode":"execution","page":1,"pageSize":100}`, recipientToken)
	require.Equal(t, http.StatusOK, personal.Code, personal.Body.String())
	require.NotContains(t, personal.Body.String(), typed.ID)
	require.Contains(t, personal.Body.String(), execution.ID)
	otherRecipientToken := login("workflow-other-manager")
	lastConfirmed := requestTask(http.MethodPatch, statusPath, `{"status":"finished"}`, otherRecipientToken)
	require.Equal(t, http.StatusOK, lastConfirmed.Code, lastConfirmed.Body.String())
	var finished dto.Assignment
	require.NoError(t, json.Unmarshal(lastConfirmed.Body.Bytes(), &finished))
	require.Equal(t, "finished", finished.Status)
	// A repeated personal confirmation is idempotent.
	require.Equal(t, http.StatusOK, requestTask(http.MethodPatch, statusPath, `{"status":"finished"}`, recipientToken).Code)

	eventRepo := repository.NewUserEventRepository(db)
	err = eventRepo.CreateFromOutbox(models.CreateUserEventRequest{
		RecipientUserID: recipientID, DocumentID: document.ID,
		DocumentKind: string(models.DocumentKindOutgoingLetter), EntityType: models.UserEventEntityAssignment,
		EventType: models.UserEventAssignmentCreated, Title: "Scoped event", Message: "Recipient only",
	}, "workflow:scoped-event")
	require.NoError(t, err)
	listed, err := eventRepo.GetList(recipientID, models.UserEventFilter{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)
	eventID := listed.Items[0].ID.String()
	queryEvents := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/user-events/query", strings.NewReader(`{"page":1,"pageSize":20}`))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		return response
	}
	recipientEvents := queryEvents(recipientToken)
	require.Equal(t, http.StatusOK, recipientEvents.Code, recipientEvents.Body.String())
	require.Contains(t, recipientEvents.Body.String(), eventID)
	outsiderEvents := queryEvents(login("workflow-outsider"))
	require.Equal(t, http.StatusOK, outsiderEvents.Code, outsiderEvents.Body.String())
	require.NotContains(t, outsiderEvents.Body.String(), eventID)

	deleteAs := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodDelete, "/api/v1/assignments/"+acknowledgment.ID, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		return response
	}
	otherManagerToken := login("workflow-other-manager")
	foreignDelete := deleteAs(otherManagerToken)
	require.Equal(t, http.StatusForbidden, foreignDelete.Code, foreignDelete.Body.String())
	var remaining int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM assignments WHERE id = $1`, acknowledgment.ID).Scan(&remaining))
	require.Equal(t, 1, remaining)
	ownerDelete := deleteAs(managerToken)
	require.Equal(t, http.StatusNoContent, ownerDelete.Code, ownerDelete.Body.String())
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM assignments WHERE id = $1`, acknowledgment.ID).Scan(&remaining))
	require.Zero(t, remaining)
	require.Equal(t, http.StatusNotFound, deleteAs(managerToken).Code)
	missingConfirm := httptest.NewRequest(http.MethodPatch, "/api/v1/assignments/"+acknowledgment.ID+"/status", strings.NewReader(`{"status":"finished"}`))
	missingConfirm.Header.Set("Authorization", "Bearer "+recipientToken)
	missingConfirmResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(missingConfirmResponse, missingConfirm)
	require.Equal(t, http.StatusNotFound, missingConfirmResponse.Code, missingConfirmResponse.Body.String())
}

type dtoAssignmentID struct {
	ID string `json:"id"`
}

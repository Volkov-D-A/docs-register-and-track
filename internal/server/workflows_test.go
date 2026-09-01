package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeAcknowledgmentAPI struct {
	createdDocumentID string
	createdUserIDs    []string
	viewedID          string
}

func (f *fakeAcknowledgmentAPI) Create(documentID, _ string, userIDs []string) (*dto.Acknowledgment, error) {
	f.createdDocumentID, f.createdUserIDs = documentID, userIDs
	return &dto.Acknowledgment{ID: uuid.NewString()}, nil
}
func (*fakeAcknowledgmentAPI) GetList(string) ([]dto.Acknowledgment, error) {
	return []dto.Acknowledgment{}, nil
}
func (*fakeAcknowledgmentAPI) GetPendingForCurrentUser() ([]dto.Acknowledgment, error) {
	return []dto.Acknowledgment{}, nil
}
func (*fakeAcknowledgmentAPI) GetCurrentUserPendingByDocument(string) ([]dto.Acknowledgment, error) {
	return []dto.Acknowledgment{}, nil
}
func (*fakeAcknowledgmentAPI) GetAllActive() ([]dto.Acknowledgment, error) {
	return []dto.Acknowledgment{}, nil
}
func (f *fakeAcknowledgmentAPI) MarkViewed(id string) error { f.viewedID = id; return nil }
func (*fakeAcknowledgmentAPI) MarkConfirmed(string) error   { return nil }
func (*fakeAcknowledgmentAPI) Delete(string) error          { return nil }

type fakeUserEventAPI struct {
	filter       models.UserEventFilter
	markedReadID string
}

func (f *fakeUserEventAPI) GetCurrentUserEvents(filter models.UserEventFilter) (*dto.PagedResult[dto.UserEvent], error) {
	f.filter = filter
	return &dto.PagedResult[dto.UserEvent]{Items: []dto.UserEvent{}}, nil
}
func (*fakeUserEventAPI) GetUnreadCount() (int, error)  { return 3, nil }
func (f *fakeUserEventAPI) MarkRead(id string) error    { f.markedReadID = id; return nil }
func (*fakeUserEventAPI) MarkDocumentRead(string) error { return nil }
func (*fakeUserEventAPI) MarkAllRead() error            { return nil }

type fakeAdministrativeOrderAcknowledgmentAPI struct {
	markedID string
}

func (f *fakeAdministrativeOrderAcknowledgmentAPI) MarkAcknowledged(id string) (*dto.AdministrativeOrderAcknowledgmentPerson, error) {
	f.markedID = id
	return &dto.AdministrativeOrderAcknowledgmentPerson{ID: id}, nil
}

func TestWorkflowAPIsRequireSessionAndUseRequestPrincipal(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	acknowledgments := &fakeAcknowledgmentAPI{}
	events := &fakeUserEventAPI{}
	var acknowledgmentPrincipal, eventPrincipal uuid.UUID
	api.acknowledgments = func(user *models.User) acknowledgmentAPI {
		acknowledgmentPrincipal = user.ID
		return acknowledgments
	}
	api.userEvents = func(user *models.User) userEventAPI {
		eventPrincipal = user.ID
		return events
	}

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/acknowledgments/pending", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	create := httptest.NewRequest(http.MethodPost, "/api/v1/acknowledgments", strings.NewReader(`{"documentId":"doc","content":"read","userIds":["user"]}`))
	create.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, create)
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	assert.Equal(t, "doc", acknowledgments.createdDocumentID)
	assert.Equal(t, []string{"user"}, acknowledgments.createdUserIDs)
	assert.NotEqual(t, uuid.Nil, acknowledgmentPrincipal)

	query := httptest.NewRequest(http.MethodPost, "/api/v1/user-events/query", strings.NewReader(`{"unreadOnly":true,"page":2,"pageSize":10}`))
	query.Header.Set("Authorization", "Bearer "+token)
	queryResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(queryResponse, query)
	require.Equal(t, http.StatusOK, queryResponse.Code, queryResponse.Body.String())
	assert.True(t, events.filter.UnreadOnly)
	assert.Equal(t, 2, events.filter.Page)
	assert.NotEqual(t, uuid.Nil, eventPrincipal)
}

func TestWorkflowMutationRoutes(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	acknowledgments := &fakeAcknowledgmentAPI{}
	events := &fakeUserEventAPI{}
	orderAcknowledgments := &fakeAdministrativeOrderAcknowledgmentAPI{}
	api.acknowledgments = func(*models.User) acknowledgmentAPI { return acknowledgments }
	api.userEvents = func(*models.User) userEventAPI { return events }
	api.administrativeOrderAcknowledgments = func(*models.User) administrativeOrderAcknowledgmentAPI { return orderAcknowledgments }

	ackID, eventID, orderPersonID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/api/v1/acknowledgments/" + ackID + "/view", http.StatusNoContent},
		{http.MethodPost, "/api/v1/user-events/" + eventID + "/read", http.StatusNoContent},
		{http.MethodPost, "/api/v1/user-events/read-all", http.StatusNoContent},
		{http.MethodPost, "/api/v1/administrative-order-acknowledgments/" + orderPersonID + "/confirm", http.StatusOK},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, req)
		require.Equal(t, tc.want, response.Code, response.Body.String())
	}
	assert.Equal(t, ackID, acknowledgments.viewedID)
	assert.Equal(t, eventID, events.markedReadID)
	assert.Equal(t, orderPersonID, orderAcknowledgments.markedID)
}

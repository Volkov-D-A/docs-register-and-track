package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

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

func TestWorkflowMutationRoutes(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	events := &fakeUserEventAPI{}
	orderAcknowledgments := &fakeAdministrativeOrderAcknowledgmentAPI{}
	api.userEvents = func(*models.User) userEventAPI { return events }
	api.administrativeOrderAcknowledgments = func(*models.User) administrativeOrderAcknowledgmentAPI { return orderAcknowledgments }

	eventID, orderPersonID := uuid.NewString(), uuid.NewString()
	for _, tc := range []struct {
		method, path string
		want         int
	}{
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
	assert.Equal(t, eventID, events.markedReadID)
	assert.Equal(t, orderPersonID, orderAcknowledgments.markedID)
}

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

type fakeAssignmentAPI struct {
	filter  models.AssignmentFilter
	created assignmentDetailsRequest
}

func (f *fakeAssignmentAPI) Create(documentID, executorID, content, deadline string, coExecutorIDs []string) (*dto.Assignment, error) {
	f.created = assignmentDetailsRequest{DocumentID: documentID, ExecutorID: executorID, Content: content, Deadline: deadline, CoExecutorIDs: coExecutorIDs}
	return &dto.Assignment{ID: uuid.NewString()}, nil
}
func (*fakeAssignmentAPI) CreateSeries(models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	return &dto.AssignmentSeries{}, nil
}
func (*fakeAssignmentAPI) GetSeries(string) (*dto.AssignmentSeries, error) {
	return &dto.AssignmentSeries{}, nil
}
func (*fakeAssignmentAPI) GetSeriesHistory(string) ([]dto.Assignment, error) {
	return []dto.Assignment{}, nil
}
func (*fakeAssignmentAPI) UpdateSeries(string, models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	return &dto.AssignmentSeries{}, nil
}
func (*fakeAssignmentAPI) CancelSeries(string) error { return nil }
func (*fakeAssignmentAPI) Update(string, string, string, string, []string) (*dto.Assignment, error) {
	return &dto.Assignment{}, nil
}
func (*fakeAssignmentAPI) UpdateStatus(string, string, string) (*dto.Assignment, error) {
	return &dto.Assignment{}, nil
}
func (*fakeAssignmentAPI) GetByID(string) (*dto.Assignment, error) { return &dto.Assignment{}, nil }
func (f *fakeAssignmentAPI) GetList(filter models.AssignmentFilter) (*dto.PagedResult[dto.Assignment], error) {
	f.filter = filter
	return &dto.PagedResult[dto.Assignment]{Items: []dto.Assignment{}}, nil
}
func (*fakeAssignmentAPI) Delete(string) error { return nil }

func TestAssignmentAPIRequiresSessionAndUsesRequestPrincipal(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	assignments := &fakeAssignmentAPI{}
	var principalID uuid.UUID
	api.assignments = func(user *models.User) assignmentAPI {
		principalID = user.ID
		return assignments
	}

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/api/v1/assignments/query", strings.NewReader(`{}`)))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	malicious := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/query", strings.NewReader(`{"search":"needle","allowedDocumentKinds":["incoming_letter"],"accessibleByUserId":"forged"}`))
	malicious.Header.Set("Authorization", "Bearer "+token)
	maliciousResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(maliciousResponse, malicious)
	require.Equal(t, http.StatusBadRequest, maliciousResponse.Code, maliciousResponse.Body.String())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/query", strings.NewReader(`{"search":"needle"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.NotEqual(t, uuid.Nil, principalID)
	assert.Equal(t, "needle", assignments.filter.Search)
	assert.Empty(t, assignments.filter.AllowedDocumentKinds)
	assert.Empty(t, assignments.filter.AccessibleByUserID)
}

func TestAssignmentAPIDecodesCoExecutors(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	assignments := &fakeAssignmentAPI{}
	api.assignments = func(*models.User) assignmentAPI { return assignments }

	request := httptest.NewRequest(http.MethodPost, "/api/v1/assignments", strings.NewReader(`{"documentId":"doc","executorId":"executor","content":"work","deadline":"2026-09-01","coExecutorIds":["co-1","co-2"]}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	assert.Equal(t, []string{"co-1", "co-2"}, assignments.created.CoExecutorIDs)
}

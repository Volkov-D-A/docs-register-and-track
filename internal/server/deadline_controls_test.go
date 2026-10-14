package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type deadlineAPIStub struct {
	deadlineControlAPI
	created models.DeadlineControlRequest
	calls   int
}

func (s *deadlineAPIStub) Create(_ context.Context, r models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
	s.created = r
	s.calls++
	return &models.DocumentDeadlineControl{ID: uuid.NewString(), DocumentID: r.DocumentID}, nil
}
func TestDeadlineHTTPContractsUseSessionAndRejectActorInjection(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	stub := &deadlineAPIStub{}
	var principalID uuid.UUID
	api.deadlines = func(user *models.User) deadlineControlAPI { principalID = user.ID; return stub }
	handler := api.Handler()
	for _, path := range []string{"/api/v1/deadline-controls", "/api/v1/deadline-controls/query", "/api/v1/deadline-series", "/api/v1/deadline-executors", "/api/v1/deadline-controls/control-id/complete", "/api/v1/deadline-series/series-id/resume"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`)))
		require.Equal(t, http.StatusUnauthorized, response.Code)
	}
	for _, body := range []string{`{"createdBy":"forged"}`, `{"completedBy":"forged"}`, `{"canManage":true}`, `{"documentAccessScope":{"restricted":false}}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/deadline-controls", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	}
	require.Zero(t, stub.calls)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/deadline-controls", strings.NewReader(`{"documentId":"document-id","executorId":"executor-id","coExecutorIds":[],"subject":"Контроль","dueDate":"2026-10-14","version":0,"changeReason":""}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.NotEqual(t, uuid.Nil, principalID)
	require.Equal(t, 1, stub.calls)
	require.Equal(t, "document-id", stub.created.DocumentID)
}

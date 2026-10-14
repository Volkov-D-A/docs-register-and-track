package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScheduledReportEndpointsAreRemoved(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	handler := api.Handler()
	for _, endpoint := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/reports/schedules"},
		{http.MethodPost, "/api/v1/reports/schedules"},
		{http.MethodDelete, "/api/v1/reports/schedules/11111111-1111-1111-1111-111111111111"},
		{http.MethodGet, "/api/v1/reports/schedules/11111111-1111-1111-1111-111111111111/runs"},
		{http.MethodGet, "/api/v1/reports/runs/11111111-1111-1111-1111-111111111111/file"},
		{http.MethodPost, "/api/v1/reports/runs/11111111-1111-1111-1111-111111111111/retry"},
	} {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			request := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(`{}`))
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
		})
	}
}

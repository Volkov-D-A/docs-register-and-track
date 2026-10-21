package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRemovedAPIsHaveNoRoutes(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	id := uuid.NewString()
	for _, route := range []struct{ method, path string }{
		{"GET", "/api/v1/users/executors"},
		{"POST", "/api/v1/user-events/" + id + "/read"},
		{"GET", "/api/v1/deadline-controls/" + id},
		{"GET", "/api/v1/documents/" + id + "/attachments"},
		{"POST", "/api/v1/documents/" + id + "/attachments"},
		{"GET", "/api/v1/attachments/" + id},
		{"GET", "/api/v1/attachments/" + id + "/download"},
		{"DELETE", "/api/v1/attachments/" + id},
		{"POST", "/api/v1/attachments/delete"},
		{"GET", "/api/v1/statistics/storage"},
		{"GET", "/api/v1/profile/substitution"},
		{"PUT", "/api/v1/profile/substitution"},
		{"GET", "/api/v1/profile/substitution-candidates"},
		{"GET", "/api/v1/users/" + id + "/substitution"},
		{"PUT", "/api/v1/users/" + id + "/substitution"},
		{"POST", "/api/v1/statistics/storage/refresh"},
		{"GET", "/api/v1/statistics/storage/status"},
		{"POST", "/api/v1/statistics/storage/retry"},
	} {
		t.Run(route.method+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, req)
			require.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, response.Code, response.Body.String())
		})
	}
}

func TestRemovedFileSettingsCannotBeReadOrWritten(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	store := &fakeSettingsManagementStore{}
	api.settings = store
	for _, key := range []string{"max_file_size_mb", "allowed_file_types", "assignment_completion_attachments_enabled"} {
		for _, method := range []string{http.MethodGet, http.MethodPatch} {
			req := httptest.NewRequest(method, "/api/v1/settings/"+key, strings.NewReader(`{"value":"15"}`))
			req.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, req)
			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		}
	}
	require.Zero(t, store.updateCalls)
	require.Zero(t, store.getAllCalls)
}

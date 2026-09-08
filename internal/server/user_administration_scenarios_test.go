package server

import (
	"errors"
	"fmt"
	"github.com/lib/pq"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserAdministrationRejectsUnprivilegedRequests(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, nil)
	access := &fakeUserAccessManagementStore{}
	api.userAccess = access
	id := users.users[0].ID.String()
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users"}, {http.MethodPost, "/api/v1/users"},
		{http.MethodPatch, "/api/v1/users/" + id}, {http.MethodPost, "/api/v1/users/" + id + "/reset-password"},
		{http.MethodGet, "/api/v1/users/" + id + "/access-profile"}, {http.MethodPut, "/api/v1/users/" + id + "/access-profile"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			for _, bearer := range []string{"", token} {
				request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
				if bearer != "" {
					request.Header.Set("Authorization", "Bearer "+bearer)
				}
				response := httptest.NewRecorder()
				api.Handler().ServeHTTP(response, request)
				status := http.StatusUnauthorized
				if bearer != "" {
					status = http.StatusForbidden
				}
				require.Equal(t, status, response.Code, response.Body.String())
				require.Empty(t, users.effects)
				require.Empty(t, access.effects)
			}
		})
	}
}

func TestUserAdministrationRejectsInvalidAndMissingTargets(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	access := &fakeUserAccessManagementStore{}
	api.userAccess = access
	for _, tc := range []struct {
		id     string
		status int
	}{{"invalid", http.StatusBadRequest}, {uuid.NewString(), http.StatusNotFound}} {
		for _, route := range []struct{ method, suffix string }{{http.MethodGet, "/access-profile"}, {http.MethodPut, "/access-profile"}, {http.MethodPost, "/reset-password"}} {
			request := httptest.NewRequest(route.method, "/api/v1/users/"+tc.id+route.suffix, strings.NewReader(`{}`))
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			require.Empty(t, users.effects)
			require.Empty(t, access.effects)
		}
	}
}

func TestUserAccessProfileReadUsesTarget(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	profile := &models.UserDocumentAccessProfile{SystemPermissions: []models.UserSystemPermissionRule{{Permission: models.SystemPermissionReferences, IsAllowed: true}}}
	api.userAccess = &fakeUserAccessManagementStore{profile: profile}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+users.users[0].ID.String()+"/access-profile", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), models.SystemPermissionReferences)
}

type failingExecutorStore struct{}

func (failingExecutorStore) GetExecutors() ([]models.User, error) {
	return nil, errors.New("executors unavailable")
}
func TestExecutorLookupRejectsDeactivatedSessionAndReportsStoreError(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	api.executors = failingExecutorStore{}
	for _, active := range []bool{true, false} {
		api.authUsers.(*fakeAuthUsers).user.IsActive = active
		request := httptest.NewRequest(http.MethodGet, "/api/v1/users/executors", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		status := http.StatusUnauthorized
		if active {
			status = http.StatusInternalServerError
		}
		require.Equal(t, status, response.Code, response.Body.String())
	}
}

type failedAdminUpdateStore struct {
	*fakeUserManagementStore
	err error
}

func (s failedAdminUpdateStore) UpdateWithOutbox(models.UpdateUserRequest, []models.OutboxEvent) (*models.User, error) {
	return nil, s.err
}
func TestUserUpdateReportsLastAdministratorConflict(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{fmt.Errorf("commit: %w", &pq.Error{Code: "P0001", Message: "at least one active administrator must remain"}), http.StatusConflict},
		{models.NewForbidden("нет доступа"), http.StatusForbidden},
	} {
		api, users, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
		api.userCommands = failedAdminUpdateStore{users, tc.err}
		request := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+users.users[0].ID.String(), strings.NewReader(`{"login":"target","fullName":"Target","isActive":false}`))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, tc.status, response.Code, response.Body.String())
		require.Empty(t, users.effects)
	}
}

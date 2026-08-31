package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestProfileAPIUpdatesBearerPrincipalWithAtomicAudit(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	actor := api.authUsers.(*fakeAuthUsers).user
	users.users = append(users.users, *actor)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/profile", strings.NewReader(`{"login":"renamed","fullName":"Renamed User"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	updated, err := users.GetByID(actor.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "renamed", updated.Login)
	assert.Equal(t, "Renamed User", updated.FullName)
	require.Len(t, users.effects, 1)
	assert.Contains(t, users.effects[0].Payload, "USER_PROFILE_UPDATE")
}

func TestProfileAPISubstitutionCandidatesReturnOnlyActiveUsers(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, nil)
	users.users = append(users.users,
		models.User{ID: uuid.New(), Login: "active", FullName: "Active", IsActive: true},
		models.User{ID: uuid.New(), Login: "inactive", FullName: "Inactive", IsActive: false},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/profile/substitution-candidates", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var result []map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
	for _, user := range result {
		assert.NotEqual(t, "inactive", user["login"])
	}
}

func TestProfileAPISelfSubstitutionUsesSessionUserAndAudit(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, nil)
	actor := api.authUsers.(*fakeAuthUsers).user
	departmentID := uuid.New()
	actor.IsDocumentParticipant = true
	actor.DepartmentID = &departmentID
	substitute := models.User{ID: uuid.New(), Login: "substitute", FullName: "Substitute", IsActive: true, DepartmentID: &departmentID}
	users.users = append(users.users, substitute)
	store := &fakeUserSubstitutionManagementStore{}
	api.substitutions = store
	request := httptest.NewRequest(http.MethodPut, "/api/v1/profile/substitution", strings.NewReader(`{"principalUserId":"`+uuid.NewString()+`","substituteUserId":"`+substitute.ID.String()+`","isActive":true}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, actor.ID, store.principalID)
	require.NotNil(t, store.createdBy)
	assert.Equal(t, actor.ID, *store.createdBy)
	require.Len(t, store.effects, 1)
	assert.Contains(t, store.effects[0].Payload, "USER_SUBSTITUTION_SELF_UPDATE")
}

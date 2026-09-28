package server

import (
	"encoding/json"
	"errors"
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
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/profile", strings.NewReader(`{"login":"renamed","lastName":"Renamed","firstName":"User","noPatronymic":true}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	updated, err := users.GetByID(actor.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "renamed", updated.Login)
	assert.Equal(t, "Renamed User", updated.FullName())
	require.Len(t, users.effects, 1)
	assert.Contains(t, users.effects[0].Payload, "USER_PROFILE_UPDATE")
}

func TestProfileAPISubstitutionCandidatesReturnOnlyActiveUsers(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, nil)
	users.users = append(users.users,
		models.User{ID: uuid.New(), Login: "active", LastName: "Active", FirstName: "User", Patronymic: "", NoPatronymic: true, IsActive: true},
		models.User{ID: uuid.New(), Login: "inactive", LastName: "Inactive", FirstName: "User", Patronymic: "", NoPatronymic: true, IsActive: false},
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
	substitute := models.User{ID: uuid.New(), Login: "substitute", LastName: "Substitute", FirstName: "User", Patronymic: "", NoPatronymic: true, IsActive: true, DepartmentID: &departmentID}
	users.users = append(users.users, substitute)
	store := &fakeUserSubstitutionManagementStore{}
	api.substitutions = store
	request := httptest.NewRequest(http.MethodPut, "/api/v1/profile/substitution", strings.NewReader(`{"principalUserId":"`+uuid.NewString()+`","substituteUserId":"`+substitute.ID.String()+`","isActive":true}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, actor.ID, store.principalID)
	require.Len(t, store.effects, 1)
	assert.Contains(t, store.effects[0].Payload, "USER_SUBSTITUTION_SELF_UPDATE")
}

type failingProfileStore struct{ *fakeUserManagementStore }

func (s failingProfileStore) UpdateProfileWithOutbox(uuid.UUID, models.UpdateProfileRequest, []models.OutboxEvent) error {
	return errors.New("profile storage unavailable")
}
func TestProfileAPIReportsStorageFailure(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, nil)
	api.userCommands = failingProfileStore{users}
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/profile", strings.NewReader(`{"login":"renamed","lastName":"Renamed","firstName":"User","noPatronymic":true}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
	require.Empty(t, users.effects)
}

func TestProfileNameNormalizationAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"without patronymic", `{"login":" renamed ","lastName":" Иванов ","firstName":" Иван ","patronymic":"Ignored","noPatronymic":true}`, http.StatusOK},
		{"with patronymic", `{"login":"renamed","lastName":"Иванов","firstName":"Иван","patronymic":" Иванович "}`, http.StatusOK},
		{"missing patronymic", `{"login":"renamed","lastName":"Иванов","firstName":"Иван","patronymic":" "}`, http.StatusBadRequest},
		{"legacy input", `{"login":"renamed","fullName":"Иванов Иван Иванович"}`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, users, token := authenticatedUserAPI(t, nil)
			actor := api.authUsers.(*fakeAuthUsers).user
			users.users = append(users.users, *actor)
			request := httptest.NewRequest(http.MethodPatch, "/api/v1/profile", strings.NewReader(tc.body))
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status != http.StatusOK {
				require.Empty(t, users.effects)
				return
			}
			var user map[string]any
			require.NoError(t, json.NewDecoder(response.Body).Decode(&user))
			assert.Equal(t, "renamed", user["login"])
			assert.Equal(t, "Иванов", user["lastName"])
			assert.Equal(t, "Иван", user["firstName"])
			if tc.name == "without patronymic" {
				assert.Equal(t, "", user["patronymic"])
				assert.Equal(t, "Иванов Иван", user["fullName"])
			} else {
				assert.Equal(t, "Иванович", user["patronymic"])
				assert.Equal(t, "Иванов Иван Иванович", user["fullName"])
			}
		})
	}
}

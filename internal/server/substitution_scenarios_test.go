package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubstitutionProductionScenarios(t *testing.T) {
	for _, admin := range []bool{false, true} {
		scope := "self"
		if admin {
			scope = "admin"
		}
		t.Run(scope, func(t *testing.T) {
			for _, scenario := range []string{"same department non-participant substitute", "other department", "inactive substitute", "self substitution", "principal without department", "principal not participant", "missing substitute", "invalid dates", "clear without participant"} {
				t.Run(scenario, func(t *testing.T) {
					permissions := []string{}
					if admin {
						permissions = append(permissions, models.SystemPermissionAdmin)
					}
					api, users, token := authenticatedUserAPI(t, permissions)
					actor := api.authUsers.(*fakeAuthUsers).user
					principal := actor
					if admin {
						principal = &models.User{ID: uuid.New(), FullName: "Principal", IsActive: true}
					}
					department := uuid.New()
					principal.DepartmentID = &department
					principal.IsDocumentParticipant = true
					substitute := models.User{ID: uuid.New(), IsActive: true, DepartmentID: &department, IsDocumentParticipant: false}
					req := models.UpdateUserSubstitutionRequest{PrincipalUserID: uuid.NewString(), SubstituteUserID: substitute.ID.String(), StartsAt: "2026-06-01", EndsAt: "2026-06-10", IsActive: true}
					expected := http.StatusBadRequest
					switch scenario {
					case "same department non-participant substitute":
						expected = http.StatusOK
					case "other department":
						other := uuid.New()
						substitute.DepartmentID = &other
					case "inactive substitute":
						substitute.IsActive = false
					case "self substitution":
						req.SubstituteUserID = principal.ID.String()
					case "principal without department":
						principal.DepartmentID = nil
					case "principal not participant":
						principal.IsDocumentParticipant = false
					case "missing substitute":
						req.SubstituteUserID = uuid.NewString()
						expected = http.StatusNotFound
					case "invalid dates":
						req.StartsAt = "2026-07-01"
					case "clear without participant":
						principal.IsDocumentParticipant = false
						principal.DepartmentID = nil
						req.SubstituteUserID = ""
						expected = http.StatusOK
					}
					users.users = []models.User{*principal, substitute}
					store := &fakeUserSubstitutionManagementStore{}
					api.substitutions = store
					path := "/api/v1/profile/substitution"
					action := "USER_SUBSTITUTION_SELF_UPDATE"
					if admin {
						path = "/api/v1/users/" + principal.ID.String() + "/substitution"
						action = "USER_SUBSTITUTION_UPDATE"
					}
					payload, err := json.Marshal(req)
					require.NoError(t, err)
					request := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
					request.Header.Set("Authorization", "Bearer "+token)
					response := httptest.NewRecorder()
					api.Handler().ServeHTTP(response, request)
					require.Equal(t, expected, response.Code, response.Body.String())
					if expected != http.StatusOK {
						assert.Empty(t, store.effects)
						assert.Equal(t, uuid.Nil, store.principalID)
						return
					}
					assert.Equal(t, principal.ID, store.principalID)
					require.NotNil(t, store.createdBy)
					assert.Equal(t, actor.ID, *store.createdBy)
					require.Len(t, store.effects, 1)
					assert.Equal(t, models.OutboxEventAudit, store.effects[0].EventType)
					var audit models.CreateAdminAuditLogRequest
					require.NoError(t, json.Unmarshal([]byte(store.effects[0].Payload), &audit))
					assert.Equal(t, actor.ID, audit.UserID)
					assert.Equal(t, action, audit.Action)
					if req.SubstituteUserID == "" {
						assert.Nil(t, store.substituteID)
						assert.Nil(t, store.startsAt)
						assert.Nil(t, store.endsAt)
						assert.False(t, store.isActive)
						assert.JSONEq(t, "null", response.Body.String())
					} else {
						require.NotNil(t, store.substituteID)
						assert.Equal(t, substitute.ID, *store.substituteID)
						require.NotNil(t, store.startsAt)
						assert.Equal(t, req.StartsAt, store.startsAt.Format("2006-01-02"))
						require.NotNil(t, store.endsAt)
						assert.Equal(t, req.EndsAt, store.endsAt.Format("2006-01-02"))
						assert.True(t, store.isActive)
						var result dto.UserSubstitution
						require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
						assert.Equal(t, substitute.ID.String(), result.SubstituteUserID)
					}
				})
			}
		})
	}
}

func TestSubstitutionProductionAuthorization(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, nil)
	store := &fakeUserSubstitutionManagementStore{}
	api.substitutions = store
	for _, tc := range []struct {
		path, token string
		status      int
	}{
		{"/api/v1/profile/substitution", "", http.StatusUnauthorized},
		{"/api/v1/users/" + users.users[0].ID.String() + "/substitution", token, http.StatusForbidden},
	} {
		request := httptest.NewRequest(http.MethodPut, tc.path, bytes.NewBufferString(`{}`))
		if tc.token != "" {
			request.Header.Set("Authorization", "Bearer "+tc.token)
		}
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, tc.status, response.Code, response.Body.String())
		assert.Empty(t, store.effects)
	}
}

package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type summaryAccessStore struct {
	fakeUserAccessManagementStore
	clerk, admin bool
}

func (s *summaryAccessStore) HasPermission(_, action, _, _ string) (bool, error) {
	return s.clerk && (action == string(models.DocumentActionRead) || action == string(models.DocumentActionCreate) || action == string(models.DocumentActionAssign)), nil
}
func (s *summaryAccessStore) HasSystemPermission(permission, _ string) (bool, error) {
	return s.admin && permission == models.SystemPermissionAdmin, nil
}

func TestCurrentAccessSummaryProductionScenarios(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		clerk, participant, admin bool
	}{
		{"clerk", true, false, false}, {"participant without full read", false, true, false}, {"admin without document permissions", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			permissions := []string{}
			if tc.admin {
				permissions = append(permissions, models.SystemPermissionAdmin)
			}
			api, _, token := authenticatedUserAPI(t, permissions)
			api.authUsers.(*fakeAuthUsers).user.IsDocumentParticipant = tc.participant
			api.userAccess = &summaryAccessStore{clerk: tc.clerk, admin: tc.admin}
			api.substitutions = &fakeUserSubstitutionManagementStore{}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/access/current", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var summary dto.CurrentAccessSummary
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &summary))
			documentAccess := tc.clerk || tc.participant
			assert.Equal(t, documentAccess, summary.DocumentDomainAccess)
			assert.Equal(t, documentAccess, summary.Sections.Dashboard)
			assert.Equal(t, documentAccess, summary.Sections.Assignments)
			assert.Equal(t, documentAccess, summary.Sections.Incoming)
			assert.Equal(t, documentAccess, summary.Sections.Outgoing)
			assert.Equal(t, documentAccess, summary.Sections.Appeals)
			assert.Equal(t, documentAccess, summary.Sections.Orders)
			assert.Equal(t, tc.admin, summary.Sections.Settings)
			require.Len(t, summary.DocumentKinds, 4)
			for _, kind := range summary.DocumentKinds {
				assert.Equal(t, tc.clerk, kind.CanRegister)
				assert.Equal(t, tc.clerk, kind.CanReadFull)
			}
			if tc.clerk {
				assert.ElementsMatch(t, []string{"incoming_letter", "outgoing_letter", "citizen_appeal", "administrative_order"}, summary.RegistrationKinds)
			} else {
				assert.Empty(t, summary.RegistrationKinds)
			}
			assert.ElementsMatch(t, permissions, summary.SystemPermissions)
		})
	}
}

func TestCurrentAccessSummaryMaintenanceUsesHTTPGate(t *testing.T) {
	for _, admin := range []bool{false, true} {
		t.Run(map[bool]string{false: "non-admin", true: "admin"}[admin], func(t *testing.T) {
			permissions := []string{}
			if admin {
				permissions = append(permissions, models.SystemPermissionAdmin)
			}
			api, _, token := authenticatedUserAPI(t, permissions)
			api.lifecycle = &fakeManagementLifecycle{readyErr: errors.New("schema maintenance")}
			api.migrations = &fakeManagementMigrations{status: dto.MigrationStatus{CurrentVersion: 7, LatestAvailableVersion: 8, Compatible: true}}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/access/current", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, request)
			require.Equal(t, http.StatusServiceUnavailable, response.Code)
			assert.Contains(t, response.Body.String(), `"code":"maintenance"`)
			response = httptest.NewRecorder()
			api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/migrations", nil))
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			assert.Contains(t, response.Body.String(), `"currentVersion":7`)
		})
	}
}

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestCurrentAccessSummaryMatchesIndividualPermissionChecksIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	departmentID, userID, adminID, otherID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	hash, err := security.HashPassword("AccessPassw0rd!")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO departments (id, name) VALUES ($1, 'Access Department')`, departmentID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, last_name, first_name, no_patronymic, department_id, password_change_required) VALUES
		($1, 'access-reader', $5, 'Access', 'Reader', TRUE, $4, FALSE),
		($2, 'access-admin', $5, 'Access', 'Admin', TRUE, NULL, FALSE),
		($3, 'access-other', $5, 'Access', 'Other', TRUE, NULL, FALSE)`, userID, adminID, otherID, departmentID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES
		($1, 'references', TRUE), ($1, 'admin', FALSE), ($2, 'admin', TRUE)`, userID, adminID)
	require.NoError(t, err)
	for _, rule := range []struct {
		kind, subjectType, subjectKey, action string
		allowed                               bool
	}{
		{"incoming_letter", "department", departmentID.String(), "control_deadlines", true},
		{"incoming_letter", "user", userID.String(), "control_deadlines", false},
		{"incoming_letter", "user", userID.String(), "register", true},
		{"incoming_letter", "role", "clerk", "register", true},
		{"outgoing_letter", "user", userID.String(), "register", true},
		{"citizen_appeal", "department", departmentID.String(), "control_deadlines", false},
		{"administrative_order", "user", otherID.String(), "register", true},
	} {
		_, err = db.Exec(`INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed) VALUES ($1, $2, $3, $4, $5)`, rule.kind, rule.subjectType, rule.subjectKey, rule.action, rule.allowed)
		require.NoError(t, err)
	}
	api := newIntegrationManagementAPI(t, &App{db: db, cfg: validConfig()})
	getSummary := func(login string) dto.CurrentAccessSummary {
		t.Helper()
		loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"`+login+`","password":"AccessPassw0rd!"}`))
		loginResult := httptest.NewRecorder()
		api.Handler().ServeHTTP(loginResult, loginRequest)
		require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
		var session struct {
			AccessToken string `json:"accessToken"`
		}
		require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&session))
		request := httptest.NewRequest(http.MethodGet, "/api/v1/access/current", nil)
		request.Header.Set("Authorization", "Bearer "+session.AccessToken)
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var summary dto.CurrentAccessSummary
		require.NoError(t, json.NewDecoder(response.Body).Decode(&summary))
		return summary
	}

	readerSummary := getSummary("access-reader")
	require.Equal(t, []string{models.SystemPermissionReferences}, readerSummary.SystemPermissions)
	require.True(t, readerSummary.Sections.References)
	require.True(t, readerSummary.Sections.Statistics)
	require.True(t, readerSummary.Sections.Reports)
	access := repository.NewDocumentAccessRepository(db)
	require.Len(t, readerSummary.DocumentKinds, len(models.AllDocumentKindSpecs()))
	for i, spec := range models.AllDocumentKindSpecs() {
		kind := readerSummary.DocumentKinds[i]
		require.Equal(t, string(spec.Code), kind.Code)
		expected := make([]string, 0)
		for _, action := range spec.SupportedActions {
			allowed, err := access.HasPermission(string(spec.Code), string(action), departmentID.String(), userID.String())
			require.NoError(t, err)
			if allowed {
				expected = append(expected, string(action))
			}
		}
		require.Equal(t, expected, kind.AvailableActions)
	}
	require.Equal(t, []string{"register", "control_deadlines"}, readerSummary.DocumentKinds[0].AvailableActions)
	require.Equal(t, []string{"register"}, readerSummary.DocumentKinds[1].AvailableActions)

	adminSummary := getSummary("access-admin")
	require.Equal(t, []string{models.SystemPermissionAdmin}, adminSummary.SystemPermissions)
	require.True(t, adminSummary.Sections.Settings)
	for _, kind := range adminSummary.DocumentKinds {
		require.Empty(t, kind.AvailableActions)
	}
}

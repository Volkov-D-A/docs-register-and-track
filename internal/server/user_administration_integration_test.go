package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestUserAdministrationAPIPersistsAccessAndSubstitutionWithAuditIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	outbox := repository.NewOutboxRepository(db)
	users := repository.NewUserRepository(db)
	users.SetOutbox(outbox)
	access := repository.NewDocumentAccessRepository(db)
	access.SetOutbox(outbox)
	substitutions := repository.NewUserSubstitutionRepository(db)
	substitutions.SetOutbox(outbox)
	sessions := repository.NewServerSessionRepository(db)
	hash, err := security.HashPassword("AdminPassw0rd!")
	require.NoError(t, err)
	adminID, targetID, substituteID, departmentID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO departments (id, name) VALUES ($1, 'Integration Department')`, departmentID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, is_document_participant, department_id, password_change_required) VALUES
		($1, 'access-admin', $4, 'Access Admin', TRUE, FALSE, NULL, FALSE),
		($2, 'access-target', 'hash', 'Access Target', TRUE, TRUE, $5, FALSE),
		($3, 'access-substitute', 'hash', 'Access Substitute', TRUE, FALSE, $5, FALSE)`, adminID, targetID, substituteID, hash, departmentID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, $2, TRUE)`, adminID, models.SystemPermissionAdmin)
	require.NoError(t, err)

	api := &managementAPI{
		cfg:       &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers: users, authSettings: repository.NewSettingsRepository(db), sessions: sessions,
		audit: repository.NewAdminAuditLogRepository(db), userCommands: users,
		userAccess: access, substitutions: substitutions, departments: repository.NewDepartmentRepository(db),
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"access-admin","password":"AdminPassw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
	var loginBody struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&loginBody))
	profileRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/profile", strings.NewReader(`{"login":"access-admin-renamed","fullName":"Access Admin Renamed"}`))
	profileRequest.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	profileResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(profileResult, profileRequest)
	require.Equal(t, http.StatusNoContent, profileResult.Code, profileResult.Body.String())

	accessRequest := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+targetID.String()+"/access-profile", strings.NewReader(`{"systemPermissions":[{"permission":"references","isAllowed":true}],"permissions":[{"kindCode":"incoming_letter","action":"read","isAllowed":true}]}`))
	accessRequest.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	accessResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(accessResult, accessRequest)
	require.Equal(t, http.StatusNoContent, accessResult.Code, accessResult.Body.String())

	substitutionRequest := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+targetID.String()+"/substitution", strings.NewReader(`{"substituteUserId":"`+substituteID.String()+`","isActive":true}`))
	substitutionRequest.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	substitutionResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(substitutionResult, substitutionRequest)
	require.Equal(t, http.StatusOK, substitutionResult.Code, substitutionResult.Body.String())

	var systemRules, documentRules, substitutionsCount, effects int
	var profileLogin string
	require.NoError(t, db.QueryRow(`SELECT login FROM users WHERE id=$1`, adminID).Scan(&profileLogin))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM user_system_permissions WHERE user_id=$1 AND permission='references' AND is_allowed=TRUE`, targetID).Scan(&systemRules))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM document_permissions WHERE subject_type='user' AND subject_key=$1 AND kind_code='incoming_letter' AND action='read' AND is_allowed=TRUE`, targetID.String()).Scan(&documentRules))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM user_substitutions WHERE principal_user_id=$1 AND substitute_user_id=$2 AND is_active=TRUE`, targetID, substituteID).Scan(&substitutionsCount))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE processed_at IS NULL AND event_type=$1`, models.OutboxEventAudit).Scan(&effects))
	require.Equal(t, 1, systemRules)
	require.Equal(t, 1, documentRules)
	require.Equal(t, 1, substitutionsCount)
	require.Equal(t, "access-admin-renamed", profileLogin)
	require.Equal(t, 3, effects)
}

func TestDepartmentAPIPersistsCRUDWithAuditOutboxIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	outbox := repository.NewOutboxRepository(db)
	users := repository.NewUserRepository(db)
	users.SetOutbox(outbox)
	departments := repository.NewDepartmentRepository(db)
	departments.SetOutbox(outbox)
	sessions := repository.NewServerSessionRepository(db)
	hash, err := security.HashPassword("AdminPassw0rd!")
	require.NoError(t, err)
	adminID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active, password_change_required)
		VALUES ($1, 'department-admin', $2, 'Department Admin', TRUE, FALSE)`, adminID, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, $2, TRUE)`, adminID, models.SystemPermissionAdmin)
	require.NoError(t, err)

	api := &managementAPI{
		cfg:       &config.Config{Server: config.ServerConfig{SessionTTLHours: 12}},
		authUsers: users, authSettings: repository.NewSettingsRepository(db), sessions: sessions,
		departments: departments,
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"department-admin","password":"AdminPassw0rd!"}`))
	loginResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code, loginResult.Body.String())
	var loginBody struct {
		AccessToken string `json:"accessToken"`
	}
	require.NoError(t, json.NewDecoder(loginResult.Body).Decode(&loginBody))

	create := httptest.NewRequest(http.MethodPost, "/api/v1/departments", strings.NewReader(`{"name":"Legal","nomenclatureIds":[]}`))
	create.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	createResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResult, create)
	require.Equal(t, http.StatusCreated, createResult.Code, createResult.Body.String())
	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(createResult.Body).Decode(&created))
	departmentID := uuid.MustParse(created.ID)

	update := httptest.NewRequest(http.MethodPatch, "/api/v1/departments/"+created.ID, strings.NewReader(`{"name":"Compliance","nomenclatureIds":[]}`))
	update.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	updateResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(updateResult, update)
	require.Equal(t, http.StatusOK, updateResult.Code, updateResult.Body.String())
	var name string
	require.NoError(t, db.QueryRow(`SELECT name FROM departments WHERE id=$1`, departmentID).Scan(&name))
	require.Equal(t, "Compliance", name)

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/departments/"+created.ID, nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+loginBody.AccessToken)
	deleteResult := httptest.NewRecorder()
	api.Handler().ServeHTTP(deleteResult, deleteRequest)
	require.Equal(t, http.StatusNoContent, deleteResult.Code, deleteResult.Body.String())
	var departmentsCount, effects int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM departments WHERE id=$1`, departmentID).Scan(&departmentsCount))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE processed_at IS NULL AND event_type=$1`, models.OutboxEventAudit).Scan(&effects))
	require.Zero(t, departmentsCount)
	require.Equal(t, 3, effects)
}

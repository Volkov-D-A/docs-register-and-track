package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeUserAccessManagementStore struct {
	profile     *models.UserDocumentAccessProfile
	userID      string
	systemRules []models.UserSystemPermissionRule
	docRules    []models.UserDocumentPermissionRule
	effects     []models.OutboxEvent
}

func (s *fakeUserAccessManagementStore) HasPermission(kind, action, departmentID, userID string) (bool, error) {
	return kind == string(models.DocumentKindIncomingLetter) && action == string(models.DocumentActionRead), nil
}
func (s *fakeUserAccessManagementStore) HasSystemPermission(permission, userID string) (bool, error) {
	return permission == models.SystemPermissionAdmin, nil
}

func (s *fakeUserAccessManagementStore) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	s.userID = userID
	return s.profile, nil
}
func (s *fakeUserAccessManagementStore) ReplaceUserAccessProfileWithOutbox(userID string, systemRules []models.UserSystemPermissionRule, docRules []models.UserDocumentPermissionRule, effects []models.OutboxEvent) error {
	s.userID, s.systemRules, s.docRules, s.effects = userID, systemRules, docRules, effects
	return nil
}

type fakeUserSubstitutionManagementStore struct {
	item         *models.UserSubstitution
	principalID  uuid.UUID
	substituteID *uuid.UUID
	startsAt     *time.Time
	endsAt       *time.Time
	isActive     bool
	createdBy    *uuid.UUID
	effects      []models.OutboxEvent
}

func (s *fakeUserSubstitutionManagementStore) GetByPrincipalID(id uuid.UUID) (*models.UserSubstitution, error) {
	s.principalID = id
	return s.item, nil
}
func (s *fakeUserSubstitutionManagementStore) GetActivePrincipalIDs(uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (s *fakeUserSubstitutionManagementStore) ReplaceForPrincipalWithOutbox(principalID uuid.UUID, substituteID *uuid.UUID, startsAt, endsAt *time.Time, isActive bool, createdBy *uuid.UUID, effects []models.OutboxEvent) (*models.UserSubstitution, error) {
	s.principalID, s.substituteID, s.startsAt, s.endsAt = principalID, substituteID, startsAt, endsAt
	s.isActive, s.createdBy, s.effects = isActive, createdBy, effects
	if substituteID == nil {
		return nil, nil
	}
	return &models.UserSubstitution{ID: uuid.New(), PrincipalUserID: principalID, SubstituteUserID: *substituteID, StartsAt: startsAt, EndsAt: endsAt, IsActive: isActive}, nil
}

type fakeDepartmentLookupStore struct{ items []models.Department }

func (s fakeDepartmentLookupStore) GetAll() ([]models.Department, error) { return s.items, nil }

func TestUserAccessAPIReplacesProfileWithAtomicAudit(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	access := &fakeUserAccessManagementStore{}
	api.userAccess = access
	targetID := users.users[0].ID
	request := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+targetID.String()+"/access-profile", strings.NewReader(`{"userId":"`+uuid.NewString()+`","systemPermissions":[{"permission":"references","isAllowed":true}],"permissions":[{"kindCode":"incoming_letter","action":"read","isAllowed":true}]}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	assert.Equal(t, targetID.String(), access.userID)
	require.Len(t, access.effects, 1)
	assert.Equal(t, models.OutboxEventAudit, access.effects[0].EventType)
}

func TestUserAccessAPIRejectsUnsupportedActionBeforePersistence(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	access := &fakeUserAccessManagementStore{}
	api.userAccess = access
	request := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+users.users[0].ID.String()+"/access-profile", strings.NewReader(`{"permissions":[{"kindCode":"incoming_letter","action":"delete","isAllowed":true}]}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, access.userID)
}

func TestUserSubstitutionAPIValidatesAndPersistsSameDepartmentSubstitute(t *testing.T) {
	api, users, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	departmentID := uuid.New()
	users.users[0].IsDocumentParticipant = true
	users.users[0].DepartmentID = &departmentID
	substitute := models.User{ID: uuid.New(), Login: "substitute", FullName: "Substitute", IsActive: true, DepartmentID: &departmentID}
	users.users = append(users.users, substitute)
	substitutions := &fakeUserSubstitutionManagementStore{}
	api.substitutions = substitutions
	request := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+users.users[0].ID.String()+"/substitution", strings.NewReader(`{"principalUserId":"`+uuid.NewString()+`","substituteUserId":"`+substitute.ID.String()+`","startsAt":"2026-09-01","endsAt":"2026-09-30","isActive":true}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NotNil(t, substitutions.substituteID)
	assert.Equal(t, substitute.ID, *substitutions.substituteID)
	assert.True(t, substitutions.isActive)
	require.NotNil(t, substitutions.createdBy)
	require.Len(t, substitutions.effects, 1)
}

func TestDepartmentLookupAPIRequiresSessionAndReturnsTypedItems(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	departmentID := uuid.New()
	api.departments = fakeDepartmentLookupStore{items: []models.Department{{ID: departmentID, Name: "Legal"}}}

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/departments", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/departments", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), departmentID.String())
	assert.Contains(t, response.Body.String(), `"name":"Legal"`)
}

func TestCurrentAccessSummaryUsesBearerPrincipalOnServer(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	api.userAccess = &fakeUserAccessManagementStore{}
	api.substitutions = &fakeUserSubstitutionManagementStore{}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/access/current", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"settings":true`)
	assert.Contains(t, response.Body.String(), `"incoming":true`)
	assert.Contains(t, response.Body.String(), `"systemPermissions":["admin"]`)
}

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeReferenceManagementStore struct {
	organizations []models.Organization
	executors     []models.ResolutionExecutor
	method        string
	query         string
	id            uuid.UUID
	targetID      uuid.UUID
	name          string
	effects       []models.OutboxEvent
}

func (s *fakeReferenceManagementStore) GetAllOrganizations() ([]models.Organization, error) {
	s.method = "list-organizations"
	return s.organizations, nil
}
func (s *fakeReferenceManagementStore) FindOrCreateOrganization(name string) (*models.Organization, error) {
	s.method, s.name = "resolve-organization", name
	return &models.Organization{ID: uuid.New(), Name: name}, nil
}
func (s *fakeReferenceManagementStore) SearchOrganizations(query string) ([]models.Organization, error) {
	s.method, s.query = "search-organizations", query
	return s.organizations, nil
}
func (s *fakeReferenceManagementStore) UpdateOrganizationWithOutbox(id uuid.UUID, name string, effects []models.OutboxEvent) error {
	s.method, s.id, s.name, s.effects = "update-organization", id, name, append([]models.OutboxEvent(nil), effects...)
	return nil
}
func (s *fakeReferenceManagementStore) DeleteOrganizationWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error {
	s.method, s.id, s.effects = "delete-organization", id, append([]models.OutboxEvent(nil), effects...)
	return nil
}
func (s *fakeReferenceManagementStore) MergeOrganizationsWithOutbox(sourceID, targetID uuid.UUID, effects []models.OutboxEvent) error {
	s.method, s.id, s.targetID, s.effects = "merge-organizations", sourceID, targetID, append([]models.OutboxEvent(nil), effects...)
	return nil
}
func (s *fakeReferenceManagementStore) GetAllResolutionExecutors() ([]models.ResolutionExecutor, error) {
	s.method = "list-executors"
	return s.executors, nil
}
func (s *fakeReferenceManagementStore) FindOrCreateResolutionExecutor(name string) (*models.ResolutionExecutor, error) {
	s.method, s.name = "resolve-executor", name
	return &models.ResolutionExecutor{ID: uuid.New(), Name: name}, nil
}
func (s *fakeReferenceManagementStore) SearchResolutionExecutors(query string) ([]models.ResolutionExecutor, error) {
	s.method, s.query = "search-executors", query
	return s.executors, nil
}
func (s *fakeReferenceManagementStore) UpdateResolutionExecutorWithOutbox(id uuid.UUID, name string, effects []models.OutboxEvent) error {
	s.method, s.id, s.name, s.effects = "update-executor", id, name, append([]models.OutboxEvent(nil), effects...)
	return nil
}
func (s *fakeReferenceManagementStore) DeleteResolutionExecutorWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error {
	s.method, s.id, s.effects = "delete-executor", id, append([]models.OutboxEvent(nil), effects...)
	return nil
}

func TestReferenceReadAndResolveAPIRequiresSession(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	store := &fakeReferenceManagementStore{
		organizations: []models.Organization{{ID: uuid.New(), Name: "Legal"}},
		executors:     []models.ResolutionExecutor{{ID: uuid.New(), Name: "Executor"}},
	}
	api.references = store

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/references/organizations", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	search := httptest.NewRequest(http.MethodGet, "/api/v1/references/organizations?query=Leg", nil)
	search.Header.Set("Authorization", "Bearer "+token)
	searchResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(searchResponse, search)
	require.Equal(t, http.StatusOK, searchResponse.Code, searchResponse.Body.String())
	assert.Equal(t, "search-organizations", store.method)
	assert.Equal(t, "Leg", store.query)

	resolve := httptest.NewRequest(http.MethodPost, "/api/v1/references/resolution-executors/resolve", strings.NewReader(`{"name":"Executor"}`))
	resolve.Header.Set("Authorization", "Bearer "+token)
	resolveResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(resolveResponse, resolve)
	require.Equal(t, http.StatusOK, resolveResponse.Code, resolveResponse.Body.String())
	assert.Equal(t, "resolve-executor", store.method)
}

func TestReferenceMutationAPIRequiresPermissionAndPersistsAuditEffects(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionReferences})
	store := &fakeReferenceManagementStore{}
	api.references = store
	organizationID, targetID, executorID := uuid.New(), uuid.New(), uuid.New()

	requests := []struct {
		method     string
		path       string
		body       string
		wantMethod string
	}{
		{http.MethodPatch, "/api/v1/references/organizations/" + organizationID.String(), `{"name":"Compliance"}`, "update-organization"},
		{http.MethodPost, "/api/v1/references/organizations/" + organizationID.String() + "/merge", `{"targetId":"` + targetID.String() + `"}`, "merge-organizations"},
		{http.MethodDelete, "/api/v1/references/organizations/" + organizationID.String(), "", "delete-organization"},
		{http.MethodPatch, "/api/v1/references/resolution-executors/" + executorID.String(), `{"name":"Chief"}`, "update-executor"},
		{http.MethodDelete, "/api/v1/references/resolution-executors/" + executorID.String(), "", "delete-executor"},
	}
	for _, tt := range requests {
		request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()

		api.Handler().ServeHTTP(response, request)

		require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
		assert.Equal(t, tt.wantMethod, store.method)
		require.Len(t, store.effects, 1)
		assert.Equal(t, models.OutboxEventAudit, store.effects[0].EventType)
	}
}

func TestReferenceMutationAPIRejectsMissingPermission(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	store := &fakeReferenceManagementStore{}
	api.references = store
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/references/organizations/"+uuid.NewString(), strings.NewReader(`{"name":"Legal"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Empty(t, store.method)
}

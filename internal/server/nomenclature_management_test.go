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

type fakeNomenclatureManagementStore struct {
	items         []models.Nomenclature
	method        string
	id            uuid.UUID
	name          string
	index         string
	year          int
	kindCode      string
	separator     string
	numberingMode string
	startNumber   int
	isActive      bool
	effects       []models.OutboxEvent
}

func (s *fakeNomenclatureManagementStore) GetAll(year int, kindCode string) ([]models.Nomenclature, error) {
	s.method, s.year, s.kindCode = "list", year, kindCode
	return s.items, nil
}
func (s *fakeNomenclatureManagementStore) GetActiveByKind(kindCode string, year int) ([]models.Nomenclature, error) {
	s.method, s.year, s.kindCode = "active", year, kindCode
	return s.items, nil
}
func (s *fakeNomenclatureManagementStore) CreateWithOutbox(name, index string, year int, kindCode, separator, numberingMode string, startNumber int, effects []models.OutboxEvent) (*models.Nomenclature, error) {
	s.method, s.name, s.index, s.year, s.kindCode = "create", name, index, year, kindCode
	s.separator, s.numberingMode, s.startNumber = separator, numberingMode, startNumber
	s.effects = append([]models.OutboxEvent(nil), effects...)
	return &models.Nomenclature{ID: uuid.New(), Name: name, Index: index, Year: year, KindCode: kindCode, Separator: separator, NumberingMode: numberingMode, NextNumber: startNumber, IsActive: true}, nil
}
func (s *fakeNomenclatureManagementStore) UpdateWithOutbox(id uuid.UUID, name, index string, year int, kindCode, separator, numberingMode string, isActive bool, effects []models.OutboxEvent) (*models.Nomenclature, error) {
	s.method, s.id, s.name, s.index, s.year, s.kindCode = "update", id, name, index, year, kindCode
	s.separator, s.numberingMode, s.isActive = separator, numberingMode, isActive
	s.effects = append([]models.OutboxEvent(nil), effects...)
	return &models.Nomenclature{ID: id, Name: name, Index: index, Year: year, KindCode: kindCode, Separator: separator, NumberingMode: numberingMode, IsActive: isActive}, nil
}
func (s *fakeNomenclatureManagementStore) DeleteWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error {
	s.method, s.id, s.effects = "delete", id, append([]models.OutboxEvent(nil), effects...)
	return nil
}

func TestNomenclatureReadAPIRequiresSession(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	store := &fakeNomenclatureManagementStore{items: []models.Nomenclature{{ID: uuid.New(), Name: "Incoming"}}}
	api.nomenclature = store

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/nomenclature", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	list := httptest.NewRequest(http.MethodGet, "/api/v1/nomenclature?year=2026&kindCode=incoming_letter", nil)
	list.Header.Set("Authorization", "Bearer "+token)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, list)
	require.Equal(t, http.StatusOK, listResponse.Code, listResponse.Body.String())
	assert.Equal(t, "list", store.method)
	assert.Equal(t, 2026, store.year)
	assert.Equal(t, "incoming_letter", store.kindCode)

	active := httptest.NewRequest(http.MethodGet, "/api/v1/nomenclature/active?kindCode=incoming_letter", nil)
	active.Header.Set("Authorization", "Bearer "+token)
	activeResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(activeResponse, active)
	require.Equal(t, http.StatusOK, activeResponse.Code, activeResponse.Body.String())
	assert.Equal(t, "active", store.method)
	assert.Equal(t, time.Now().Year(), store.year)
}

func TestNomenclatureMutationAPIRequiresAdminAndPersistsAudit(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	store := &fakeNomenclatureManagementStore{}
	api.nomenclature = store

	create := httptest.NewRequest(http.MethodPost, "/api/v1/nomenclature", strings.NewReader(`{"name":"Incoming","index":"01-01","year":2026,"kindCode":"incoming_letter","separator":"/","numberingMode":"index_and_number","startNumber":0}`))
	create.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, create)
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	assert.Equal(t, "create", store.method)
	assert.Equal(t, 1, store.startNumber)
	require.Len(t, store.effects, 1)
	assert.Equal(t, models.OutboxEventAudit, store.effects[0].EventType)

	id := uuid.New()
	update := httptest.NewRequest(http.MethodPatch, "/api/v1/nomenclature/"+id.String(), strings.NewReader(`{"name":"Incoming updated","index":"01-02","year":2026,"kindCode":"incoming_letter","separator":"-","numberingMode":"number_only","isActive":false}`))
	update.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(updateResponse, update)
	require.Equal(t, http.StatusOK, updateResponse.Code, updateResponse.Body.String())
	assert.Equal(t, "update", store.method)
	assert.Equal(t, id, store.id)
	require.Len(t, store.effects, 1)

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/nomenclature/"+id.String(), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(deleteResponse, deleteRequest)
	require.Equal(t, http.StatusNoContent, deleteResponse.Code, deleteResponse.Body.String())
	assert.Equal(t, "delete", store.method)
	require.Len(t, store.effects, 1)
}

func TestNomenclatureMutationAPIRejectsNonAdmin(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	store := &fakeNomenclatureManagementStore{}
	api.nomenclature = store
	request := httptest.NewRequest(http.MethodPost, "/api/v1/nomenclature", strings.NewReader(`{"name":"Incoming"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Empty(t, store.method)
}

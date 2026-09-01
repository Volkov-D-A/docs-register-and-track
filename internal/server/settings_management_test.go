package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeSettingsManagementStore struct {
	settings    map[string]models.SystemSetting
	getAllCalls int
	updateCalls int
	updatedKey  string
	updatedVal  string
	effects     []models.OutboxEvent
}

func (s *fakeSettingsManagementStore) Get(key string) (*models.SystemSetting, error) {
	setting := s.settings[key]
	return &setting, nil
}

func (s *fakeSettingsManagementStore) GetAll() ([]models.SystemSetting, error) {
	s.getAllCalls++
	result := make([]models.SystemSetting, 0, len(s.settings))
	for _, setting := range s.settings {
		result = append(result, setting)
	}
	return result, nil
}

func (s *fakeSettingsManagementStore) UpdateWithOutbox(key, value string, effects []models.OutboxEvent) error {
	s.updateCalls++
	s.updatedKey, s.updatedVal = key, value
	s.effects = append([]models.OutboxEvent(nil), effects...)
	setting := s.settings[key]
	setting.Key, setting.Value = key, value
	s.settings[key] = setting
	return nil
}

func TestSettingsReadAPIUsesSessionAndAdminBoundaries(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	store := &fakeSettingsManagementStore{settings: map[string]models.SystemSetting{
		"organization_name": {Key: "organization_name", Value: "Docflow"},
	}}
	api.settings = store

	item := httptest.NewRequest(http.MethodGet, "/api/v1/settings/organization_name", nil)
	item.Header.Set("Authorization", "Bearer "+token)
	itemResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(itemResponse, item)
	require.Equal(t, http.StatusOK, itemResponse.Code, itemResponse.Body.String())
	assert.Contains(t, itemResponse.Body.String(), `"value":"Docflow"`)

	list := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	list.Header.Set("Authorization", "Bearer "+token)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, list)
	assert.Equal(t, http.StatusForbidden, listResponse.Code)
	assert.Zero(t, store.getAllCalls)
}

func TestSettingsAdminAPIRejectsUnsafeAttachmentLimit(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	store := &fakeSettingsManagementStore{settings: map[string]models.SystemSetting{
		"max_file_size_mb": {Key: "max_file_size_mb", Value: "15"},
	}}
	api.settings = store
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/settings/max_file_size_mb", strings.NewReader(`{"value":"2048"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, store.updatedKey)
}

func TestSettingsAdminAPIUpdatesWithAuditAndSkipsUnchangedValue(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	store := &fakeSettingsManagementStore{settings: map[string]models.SystemSetting{
		"organization_name": {Key: "organization_name", Value: "Old"},
	}}
	api.settings = store

	list := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	list.Header.Set("Authorization", "Bearer "+token)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, list)
	require.Equal(t, http.StatusOK, listResponse.Code, listResponse.Body.String())
	assert.Equal(t, 1, store.getAllCalls)

	update := httptest.NewRequest(http.MethodPatch, "/api/v1/settings/organization_name", strings.NewReader(`{"value":"New"}`))
	update.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(updateResponse, update)
	require.Equal(t, http.StatusNoContent, updateResponse.Code, updateResponse.Body.String())
	assert.Equal(t, "organization_name", store.updatedKey)
	assert.Equal(t, "New", store.updatedVal)
	require.Len(t, store.effects, 1)
	assert.Equal(t, models.OutboxEventAudit, store.effects[0].EventType)

	unchanged := httptest.NewRequest(http.MethodPatch, "/api/v1/settings/organization_name", strings.NewReader(`{"value":"New"}`))
	unchanged.Header.Set("Authorization", "Bearer "+token)
	unchangedResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(unchangedResponse, unchanged)
	require.Equal(t, http.StatusNoContent, unchangedResponse.Code, unchangedResponse.Body.String())
	assert.Equal(t, 1, store.updateCalls)
}

func TestSettingsAPIValidatesPasswordLifetimeOnServer(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	store := &fakeSettingsManagementStore{settings: map[string]models.SystemSetting{}}
	api.settings = store

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/settings/password_lifetime_days", strings.NewReader(`{"value":"month"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Zero(t, store.updateCalls)
}

func TestSettingAuditLabel(t *testing.T) {
	assert.Equal(t, "Название организации", settingAuditLabel("organization_name", nil))
	assert.Equal(t, "Пользовательское описание", settingAuditLabel("custom", &models.SystemSetting{Description: "Пользовательское описание"}))
	assert.Equal(t, "«custom»", settingAuditLabel("custom", nil))
}

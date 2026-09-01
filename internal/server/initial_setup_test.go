package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

type fakeInitialSetupStore struct {
	count int
	hash  string
}

func (s *fakeInitialSetupStore) CountUsers() (int, error) { return s.count, nil }
func (s *fakeInitialSetupStore) CreateInitialAdmin(hash string) error {
	if s.count > 0 {
		return models.NewConflict("начальная настройка уже выполнена")
	}
	s.hash, s.count = hash, 1
	return nil
}

func TestInitialSetupAPIIsServerOwnedAndOneTime(t *testing.T) {
	store := &fakeInitialSetupStore{}
	api := &managementAPI{initialSetup: store}

	status := httptest.NewRecorder()
	api.Handler().ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/v1/auth/setup-required", nil))
	require.Equal(t, http.StatusOK, status.Code)
	var result map[string]bool
	require.NoError(t, json.NewDecoder(status.Body).Decode(&result))
	assert.True(t, result["required"])

	setup := httptest.NewRecorder()
	api.Handler().ServeHTTP(setup, httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", strings.NewReader(`{"password":"Passw0rd!"}`)))
	require.Equal(t, http.StatusNoContent, setup.Code, setup.Body.String())
	assert.True(t, security.VerifyPassword(store.hash, "Passw0rd!"))

	repeated := httptest.NewRecorder()
	api.Handler().ServeHTTP(repeated, httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", strings.NewReader(`{"password":"Passw0rd!"}`)))
	assert.Equal(t, http.StatusConflict, repeated.Code)
}

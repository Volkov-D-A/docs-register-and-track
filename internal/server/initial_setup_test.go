package server

import (
	"encoding/json"
	"errors"
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

type failingInitialSetupStore struct{ countErr, createErr error }

func (s failingInitialSetupStore) CountUsers() (int, error)        { return 0, s.countErr }
func (s failingInitialSetupStore) CreateInitialAdmin(string) error { return s.createErr }

func TestInitialSetupRejectsWeakPasswordAndReportsStorageErrors(t *testing.T) {
	store := &fakeInitialSetupStore{}
	api := &managementAPI{initialSetup: store}
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", strings.NewReader(`{"password":"123"}`)))
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Zero(t, store.count)
	require.Empty(t, store.hash)
	for _, tc := range []struct {
		method, path, body string
		store              initialSetupStore
	}{
		{http.MethodGet, "/api/v1/auth/setup-required", "", failingInitialSetupStore{countErr: errors.New("unavailable")}},
		{http.MethodPost, "/api/v1/auth/setup", `{"password":"Passw0rd!"}`, failingInitialSetupStore{createErr: errors.New("unavailable")}},
	} {
		api.initialSetup = tc.store
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		require.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
	}
}

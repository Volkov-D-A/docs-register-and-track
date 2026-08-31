package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
)

type fakeDocumentCommandAPI struct {
	operation string
	kind      string
	request   any
}

func (f *fakeDocumentCommandAPI) Register(kind string, request any) (any, error) {
	f.operation, f.kind, f.request = "register", kind, request
	return map[string]string{"id": "created"}, nil
}
func (f *fakeDocumentCommandAPI) Update(kind string, request any) (any, error) {
	f.operation, f.kind, f.request = "update", kind, request
	return map[string]string{"id": "updated"}, nil
}
func (f *fakeDocumentCommandAPI) CreateAdminDraft(kind string, request services.AdminDraftCreateRequest) (any, error) {
	f.operation, f.kind, f.request = "draft", kind, request
	return map[string]string{"id": "draft"}, nil
}

func TestDocumentCommandsRequireSessionAndIdempotencyKey(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	commands := &fakeDocumentCommandAPI{}
	api.documentCommands = func(*models.User) documentCommandAPI { return commands }

	unauthorized := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/api/v1/documents/incoming_letter", strings.NewReader(`{}`)))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	missingKeyRequest := httptest.NewRequest(http.MethodPost, "/api/v1/documents/incoming_letter", strings.NewReader(`{}`))
	missingKeyRequest.Header.Set("Authorization", "Bearer "+token)
	missingKeyResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(missingKeyResponse, missingKeyRequest)
	assert.Equal(t, http.StatusBadRequest, missingKeyResponse.Code)

	key := "5d53cc58-9437-43e4-ad12-503ced0ef22d"
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/documents/outgoing_letter/document-id", strings.NewReader(`{"content":"changed"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, "update", commands.operation)
	assert.Equal(t, "outgoing_letter", commands.kind)
	body := commands.request.(map[string]any)
	assert.Equal(t, "document-id", body["id"])
	assert.Equal(t, key, body["idempotencyKey"])
}

func TestRequestDocumentPrincipalChecksSystemPermission(t *testing.T) {
	principal := requestDocumentPrincipal{user: &models.User{IsActive: true, SystemPermissions: []string{models.SystemPermissionAdmin}}}
	require.NoError(t, principal.RequireSystemPermission(models.SystemPermissionAdmin))
	require.ErrorIs(t, principal.RequireSystemPermission(models.SystemPermissionReferences), models.ErrForbidden)
}

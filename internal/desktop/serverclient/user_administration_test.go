package serverclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestUserAdministrationClientUsesTypedEndpoints(t *testing.T) {
	userID := uuid.NewString()
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/users/"+userID+"/access-profile", r.URL.Path)
			return response(http.StatusOK, `{"systemPermissions":[],"permissions":[]}`), nil
		case 2:
			assert.Equal(t, http.MethodPut, r.Method)
			assert.Equal(t, "/api/v1/users/"+userID+"/access-profile", r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		case 3:
			assert.Equal(t, "/api/v1/users/"+userID+"/substitution", r.URL.Path)
			return response(http.StatusOK, "null"), nil
		case 4:
			assert.Equal(t, "/api/v1/departments", r.URL.Path)
			return response(http.StatusOK, `[{"id":"`+uuid.NewString()+`","name":"Legal"}]`), nil
		case 5:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/departments", r.URL.Path)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{"name":"Legal","nomenclatureIds":[]}`, string(body))
			return response(http.StatusCreated, `{"id":"`+userID+`","name":"Legal","nomenclatureIds":[]}`), nil
		case 6:
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/api/v1/departments/"+userID, r.URL.Path)
			return response(http.StatusOK, `{"id":"`+userID+`","name":"Compliance","nomenclatureIds":[]}`), nil
		case 7:
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/departments/"+userID, r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		case 8:
			assert.Equal(t, "/api/v1/access/current", r.URL.Path)
			return response(http.StatusOK, `{"sections":{"settings":true},"documentKinds":[],"registrationKinds":[],"systemPermissions":["admin"]}`), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	profile, err := client.GetUserAccessProfile(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.NoError(t, client.UpdateUserAccessProfile(context.Background(), models.UpdateUserDocumentAccessRequest{UserID: userID}))
	substitution, err := client.GetUserSubstitution(context.Background(), userID)
	require.NoError(t, err)
	assert.Nil(t, substitution)
	departments, err := client.ListDepartments(context.Background())
	require.NoError(t, err)
	require.Len(t, departments, 1)
	assert.Equal(t, "Legal", departments[0].Name)
	created, err := client.CreateDepartment(context.Background(), "Legal", []string{})
	require.NoError(t, err)
	assert.Equal(t, "Legal", created.Name)
	updated, err := client.UpdateDepartment(context.Background(), userID, "Compliance", []string{})
	require.NoError(t, err)
	assert.Equal(t, "Compliance", updated.Name)
	require.NoError(t, client.DeleteDepartment(context.Background(), userID))
	summary, err := client.GetCurrentAccessSummary(context.Background())
	require.NoError(t, err)
	assert.True(t, summary.Sections.Settings)
}

func TestDepartmentClientRejectsInvalidIDBeforeRequest(t *testing.T) {
	requests := 0
	client := userClientWithToken(t, func(*http.Request) (*http.Response, error) {
		requests++
		return response(http.StatusInternalServerError, ""), nil
	})

	updated, err := client.UpdateDepartment(context.Background(), "invalid", "Legal", nil)
	assert.Nil(t, updated)
	require.Error(t, err)
	require.Error(t, client.DeleteDepartment(context.Background(), "invalid"))
	assert.Zero(t, requests)
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

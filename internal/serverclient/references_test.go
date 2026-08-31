package serverclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferenceClientUsesTypedEndpoints(t *testing.T) {
	organizationID, targetID, executorID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/references/organizations", r.URL.Path)
			assert.Equal(t, "Leg & Co", r.URL.Query().Get("query"))
			return response(http.StatusOK, `[{"id":"`+organizationID+`","name":"Legal"}]`), nil
		case 2:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/references/organizations/resolve", r.URL.Path)
			return response(http.StatusOK, `{"id":"`+organizationID+`","name":"Legal"}`), nil
		case 3:
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/api/v1/references/organizations/"+organizationID, r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		case 4:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/references/organizations/"+organizationID+"/merge", r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		case 5:
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/references/organizations/"+organizationID, r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		case 6:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/references/resolution-executors", r.URL.Path)
			assert.Equal(t, "Exec", r.URL.Query().Get("query"))
			return response(http.StatusOK, `[{"id":"`+executorID+`","name":"Executor"}]`), nil
		case 7:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/references/resolution-executors/resolve", r.URL.Path)
			return response(http.StatusOK, `{"id":"`+executorID+`","name":"Executor"}`), nil
		case 8:
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/api/v1/references/resolution-executors/"+executorID, r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		case 9:
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/references/resolution-executors/"+executorID, r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	organizations, err := client.ListOrganizations(context.Background(), "Leg & Co")
	require.NoError(t, err)
	require.Len(t, organizations, 1)
	organization, err := client.ResolveOrganization(context.Background(), "Legal")
	require.NoError(t, err)
	assert.Equal(t, organizationID, organization.ID)
	require.NoError(t, client.UpdateOrganization(context.Background(), organizationID, "Compliance"))
	require.NoError(t, client.MergeOrganizations(context.Background(), organizationID, targetID))
	require.NoError(t, client.DeleteOrganization(context.Background(), organizationID))

	executors, err := client.ListResolutionExecutors(context.Background(), "Exec")
	require.NoError(t, err)
	require.Len(t, executors, 1)
	executor, err := client.ResolveResolutionExecutor(context.Background(), "Executor")
	require.NoError(t, err)
	assert.Equal(t, executorID, executor.ID)
	require.NoError(t, client.UpdateResolutionExecutor(context.Background(), executorID, "Chief"))
	require.NoError(t, client.DeleteResolutionExecutor(context.Background(), executorID))
}

func TestReferenceClientRejectsInvalidIDsBeforeRequest(t *testing.T) {
	requests := 0
	client := userClientWithToken(t, func(*http.Request) (*http.Response, error) {
		requests++
		return response(http.StatusInternalServerError, ""), nil
	})

	require.Error(t, client.UpdateOrganization(context.Background(), "invalid", "Legal"))
	require.Error(t, client.MergeOrganizations(context.Background(), uuid.NewString(), "invalid"))
	require.Error(t, client.UpdateResolutionExecutor(context.Background(), "invalid", "Executor"))
	assert.Zero(t, requests)
}

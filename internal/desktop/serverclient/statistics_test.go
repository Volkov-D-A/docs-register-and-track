package serverclient

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatisticsClientUsesTypedAuthenticatedEndpoints(t *testing.T) {
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, "/api/v1/workspace/overview", r.URL.Path)
			return response(http.StatusOK, `{"assignments":[]}`), nil
		case 2:
			assert.Equal(t, "/api/v1/statistics/documents", r.URL.Path)
			return response(http.StatusOK, `{"year":2026,"totalYear":0,"documentsByKindMonthly":[],"documentsByRegistrarMonthly":[]}`), nil
		case 3:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/statistics/documents/report", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "kind", body["groupBy"])
			assert.NotContains(t, body, "accessScope")
			return response(http.StatusOK, `{"rows":[],"total":0}`), nil
		case 4:
			assert.Equal(t, "/api/v1/statistics/system/storage", r.URL.Path)
			return response(http.StatusOK, `{"state":"idle","storageSize":"0 B"}`), nil
		case 5:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/statistics/system/storage/retry", r.URL.Path)
			return response(http.StatusOK, `{"state":"pending","storageSize":"0 B"}`), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.GetWorkspaceOverview(context.Background(), "execution")
	require.NoError(t, err)
	_, err = client.GetDocumentStatistics(context.Background())
	require.NoError(t, err)
	_, err = client.GetDocumentReport(context.Background(), "2026-01-01", "2026-09-01", "kind", "", "", "")
	require.NoError(t, err)
	_, err = client.GetStorageStatisticsStatus(context.Background())
	require.NoError(t, err)
	status, err := client.RetryStorageStatisticsRefresh(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "pending", string(status.State))
	assert.Equal(t, 5, requestNumber)
}

func TestWorkspaceClientPassesModes(t *testing.T) {
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/api/v1/workspace/overview":
			assert.Equal(t, "execution", r.URL.Query().Get("assignmentMode"))
			return response(http.StatusOK, `{"assignmentModes":[],"assignments":[]}`), nil
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
			return nil, nil
		}
	})
	_, err := client.GetWorkspaceOverview(context.Background(), "execution")
	require.NoError(t, err)
}

func TestWorkspaceRecentDocumentsClientUsesFixedAuthenticatedPreview(t *testing.T) {
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/workspace/documents", r.URL.Path)
		require.Empty(t, r.URL.RawQuery)
		return response(http.StatusOK, `{"available":true,"items":[{"id":"doc","documentKind":"incoming_letter","documentNumber":"125","documentDate":"2026-09-20T00:00:00Z","registeredAt":"2026-09-24T12:00:00Z","description":"Alpha","correspondents":["Alpha","Beta"]}]}`), nil
	})
	result, err := client.GetWorkspaceDocuments(context.Background())
	require.NoError(t, err)
	require.True(t, result.Available)
	require.Len(t, result.Items, 1)
	require.Equal(t, "2026-09-20", result.Items[0].DocumentDate.Format("2006-01-02"))
	require.Equal(t, "2026-09-24", result.Items[0].RegisteredAt.Format("2006-01-02"))
	require.Equal(t, []string{"Alpha", "Beta"}, result.Items[0].Correspondents)
}

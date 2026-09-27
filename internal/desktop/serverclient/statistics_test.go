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
			return response(http.StatusOK, `{"assignments":[],"acknowledgments":[]}`), nil
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

	_, err := client.GetWorkspaceOverview(context.Background(), "execution", "control")
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

func TestWorkspaceClientPassesModesAndPagination(t *testing.T) {
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/api/v1/workspace/overview":
			assert.Equal(t, "execution", r.URL.Query().Get("assignmentMode"))
			assert.Equal(t, "control", r.URL.Query().Get("acknowledgmentMode"))
			return response(http.StatusOK, `{"assignmentModes":[],"assignments":[],"acknowledgmentModes":[],"acknowledgments":[]}`), nil
		case "/api/v1/workspace/acknowledgments":
			assert.Equal(t, "control", r.URL.Query().Get("mode"))
			assert.Equal(t, "2", r.URL.Query().Get("page"))
			assert.Equal(t, "7", r.URL.Query().Get("pageSize"))
			return response(http.StatusOK, `{"items":[],"page":2,"pageSize":7}`), nil
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
			return nil, nil
		}
	})
	_, err := client.GetWorkspaceOverview(context.Background(), "execution", "control")
	require.NoError(t, err)
	_, err = client.ListWorkspaceAcknowledgments(context.Background(), "control", 2, 7)
	require.NoError(t, err)
}

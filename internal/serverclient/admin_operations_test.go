package serverclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminOperationsClientUsesTypedAuthenticatedEndpoints(t *testing.T) {
	id, requestNumber := uuid.NewString(), 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, "/api/v1/admin/audit", r.URL.Path)
			assert.Equal(t, "2", r.URL.Query().Get("page"))
			assert.Equal(t, "20", r.URL.Query().Get("pageSize"))
			return response(http.StatusOK, `{"items":[],"total":0,"page":2}`), nil
		case 2:
			assert.Equal(t, "/api/v1/admin/outbox/stats", r.URL.Path)
			return response(http.StatusOK, `{"Failed":1}`), nil
		case 3:
			assert.Equal(t, "/api/v1/admin/outbox/failed", r.URL.Path)
			assert.Equal(t, "100", r.URL.Query().Get("limit"))
			return response(http.StatusOK, `[]`), nil
		case 4:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/admin/outbox/"+id+"/requeue", r.URL.Path)
			return response(http.StatusNoContent, ``), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.GetAdminAuditLog(context.Background(), 2, 20)
	require.NoError(t, err)
	stats, err := client.GetOutboxStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, stats.Failed)
	_, err = client.GetFailedOutboxEvents(context.Background(), 100)
	require.NoError(t, err)
	require.NoError(t, client.RequeueOutboxEvent(context.Background(), id))
	assert.Equal(t, 4, requestNumber)
}

package serverclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupStatusUsesSeparateHeaderCapabilityWithoutReplacingSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/admin/backups/operations/job", r.URL.Path)
		require.Empty(t, r.URL.RawQuery)
		require.Equal(t, "Bearer status-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"id":"job","kind":"restore","state":"restoring"}`))
		require.NoError(t, err)
	}))
	defer server.Close()
	client, err := New(server.URL)
	require.NoError(t, err)
	client.token = "ordinary-session"
	op, err := client.GetBackupOperation(context.Background(), "job", "status-token")
	require.NoError(t, err)
	require.Equal(t, "restoring", op.State)
	require.Equal(t, "ordinary-session", client.token)
}

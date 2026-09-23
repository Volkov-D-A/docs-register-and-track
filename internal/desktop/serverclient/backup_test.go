package serverclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
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

func TestBackupOperationEventsUseStatusCapabilityAndStopAtTerminalState(t *testing.T) {
	type request struct {
		path, method, authorization string
	}
	requests := make(chan request, 3)
	expires := time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- request{r.URL.Path, r.Method, r.Header.Get("Authorization")}
		switch r.URL.Path {
		case "/api/v1/admin/backups/catalog/restore":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, `{"job":{"id":"job","kind":"restore","state":"queued"},"statusToken":"status-token","expiresAt":%q}`, expires)
		case "/api/v1/admin/backups/operations/job/events":
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "event: operation\ndata: {\"id\":\"job\",\"state\":\"completed\"}\n\n")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := New(server.URL)
	require.NoError(t, err)
	client.token = "ordinary-session"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan LiveEvent, 1)
	client.ConfigureEvents(ctx, func(event LiveEvent) { events <- event })
	started, err := client.StartBackupOperation(ctx, "restore", models.BackupOperationRequest{CopyID: "copy"})
	require.NoError(t, err)
	require.Equal(t, "status-token", started.StatusToken)
	select {
	case event := <-events:
		require.Equal(t, "operation", event.Topic)
		require.NotNil(t, event.Operation)
		require.Equal(t, "completed", event.Operation.State)
		raw, err := json.Marshal(event)
		require.NoError(t, err)
		require.NotContains(t, string(raw), "status-token")
	case <-time.After(3 * time.Second):
		t.Fatal("missing terminal backup operation event")
	}
	require.Equal(t, "ordinary-session", client.token)
	require.Equal(t, request{"/api/v1/admin/backups/catalog/restore", http.MethodPost, "Bearer ordinary-session"}, <-requests)
	require.Equal(t, request{"/api/v1/admin/backups/operations/job/events", http.MethodGet, "Bearer status-token"}, <-requests)
	select {
	case extra := <-requests:
		t.Fatalf("terminal operation reconnected: %+v", extra)
	case <-time.After(1200 * time.Millisecond):
	}
}

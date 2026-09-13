package serverclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReadEventsHandlesFramesAndStopsAtTerminal(t *testing.T) {
	var names []string
	err := readEvents(strings.NewReader(": keepalive\r\nevent: resync\r\ndata: null\r\n\r\nevent: operation\ndata: {\ndata: }\n\n"), func(name string, data []byte) bool {
		names = append(names, name)
		if name == "operation" {
			require.Equal(t, "{\n}", string(data))
			return false
		}
		return true
	})
	require.ErrorIs(t, err, errStopEvents)
	require.Equal(t, []string{"resync", "operation"}, names)
}

func TestSessionStreamReconnectsAndStopsWithSession(t *testing.T) {
	var connections atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/login" {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"accessToken":"private-token","user":{"id":"user-id"}}`)
			return
		}
		if r.URL.Path == "/api/v1/auth/logout" {
			w.WriteHeader(204)
			return
		}
		if r.Header.Get("Authorization") != "Bearer private-token" {
			w.WriteHeader(401)
			return
		}
		connections.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: resync\ndata: null\n\n")
		w.(http.Flusher).Flush()
		if connections.Load() > 1 {
			<-r.Context().Done()
		}
	}))
	defer srv.Close()
	c, err := NewWithOptions(srv.URL, Options{AllowInsecureHTTP: true})
	require.NoError(t, err)
	// Prove that the ordinary short request timeout does not terminate the stream.
	c.http.Timeout = 100 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan LiveEvent, 8)
	c.ConfigureEvents(ctx, func(e LiveEvent) { events <- e })
	_, err = c.Login(ctx, "user", "password")
	require.NoError(t, err)
	for range 2 {
		select {
		case e := <-events:
			require.Equal(t, "resync", e.Topic)
			require.Equal(t, c.SessionState().Revision, e.Revision)
		case <-time.After(5 * time.Second):
			t.Fatal("missing reconnect snapshot")
		}
	}
	require.NoError(t, c.Logout(ctx))
	require.Eventually(t, func() bool { return !c.SessionState().Authenticated }, time.Second, time.Millisecond)
}

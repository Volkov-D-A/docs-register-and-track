package serverclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestClientApplyUsesBasicAuthentication(t *testing.T) {
	client, err := New("https://server.test")
	require.NoError(t, err)
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		login, password, ok := r.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "admin", login)
		assert.Equal(t, "Passw0rd!", password)
		assert.Equal(t, "/api/v1/admin/migrations/apply", r.URL.Path)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"currentVersion":8,"latestAvailableVersion":8,"upToDate":true}`)), Header: make(http.Header)}, nil
	})

	status, err := client.Apply(context.Background(), "admin", "Passw0rd!")

	require.NoError(t, err)
	assert.EqualValues(t, 8, status.CurrentVersion)
}

func TestClientRollbackDoesNotPutPasswordInJSON(t *testing.T) {
	client, err := New("https://server.test")
	require.NoError(t, err)
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.NotContains(t, body, "password")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"currentVersion":7}`)), Header: make(http.Header)}, nil
	})

	_, err = client.Rollback(context.Background(), "admin", "secret", models.RollbackMigrationRequest{
		BackupCompleted:      true,
		BackupReference:      "backup-1",
		AcknowledgedDataLoss: true,
		Confirmation:         "ОТКАТ МИГРАЦИИ",
		Password:             "secret",
	})

	require.NoError(t, err)
}

func TestNewRejectsPlainHTTPForRemoteServer(t *testing.T) {
	_, err := New("http://docflow.internal:8080")

	require.EqualError(t, err, "server URL must use https unless it points to localhost")
}

func TestNewAllowsPlainHTTPForLocalDevelopment(t *testing.T) {
	_, err := New("http://127.0.0.1:8080")

	require.NoError(t, err)
}

func TestNewAllowsRemoteHTTPOnlyWithExplicitOptIn(t *testing.T) {
	_, err := NewWithOptions("http://docflow.internal:8080", Options{AllowInsecureHTTP: true})

	require.NoError(t, err)
}

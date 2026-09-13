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

func TestAuthClientStoresTokenForSubsequentRequests(t *testing.T) {
	client, err := New("https://server.test")
	require.NoError(t, err)
	requestNumber := 0
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestNumber++
		if requestNumber == 1 {
			assert.Equal(t, "/api/v1/auth/login", r.URL.Path)
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accessToken":"token-1","expiresAt":"2030-01-01T00:00:00Z","user":{"id":"00000000-0000-0000-0000-000000000001","login":"admin"}}`)), Header: make(http.Header)}, nil
		}
		assert.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","login":"admin"}`)), Header: make(http.Header)}, nil
	})

	user, err := client.Login(context.Background(), "admin", "Passw0rd!")
	require.NoError(t, err)
	assert.Equal(t, "admin", user.Login)
	_, err = client.Me(context.Background())
	require.NoError(t, err)
}

func TestAuthClientChangePasswordClearsSessionAfterSuccess(t *testing.T) {
	client, err := New("https://server.test")
	require.NoError(t, err)
	requestNumber := 0
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestNumber++
		switch requestNumber {
		case 1:
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accessToken":"token-1","expiresAt":"2030-01-01T00:00:00Z","user":{"id":"00000000-0000-0000-0000-000000000001","login":"admin"}}`)), Header: make(http.Header)}, nil
		case 2:
			assert.Equal(t, "/api/v1/auth/change-password", r.URL.Path)
			assert.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "Passw0rd!", body["oldPassword"])
			assert.Equal(t, "NewPassw0rd!", body["newPassword"])
			return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
		default:
			t.Fatalf("unexpected HTTP request %d", requestNumber)
			return nil, nil
		}
	})

	_, err = client.Login(context.Background(), "admin", "Passw0rd!")
	require.NoError(t, err)
	require.NoError(t, client.ChangePassword(context.Background(), "Passw0rd!", "NewPassw0rd!"))
	_, err = client.Me(context.Background())
	assert.ErrorIs(t, err, models.ErrUnauthorized)
	assert.Equal(t, 2, requestNumber)
}

func TestAuthClientChangeRequiredPasswordDoesNotRequireSession(t *testing.T) {
	client, err := New("https://server.test")
	require.NoError(t, err)
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v1/auth/change-required-password", r.URL.Path)
		assert.Empty(t, r.Header.Get("Authorization"))
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})

	err = client.ChangeRequiredPassword(context.Background(), "admin", "Passw0rd!", "NewPassw0rd!")
	require.NoError(t, err)
}

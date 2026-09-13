package serverclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func userClientWithToken(t *testing.T, transport roundTripFunc) *Client {
	t.Helper()
	client, err := New("https://server.test")
	require.NoError(t, err)
	client.token = "session-token"
	client.http.Transport = transport
	return client
}

func TestUserClientUsesBearerAndTypedListResponse(t *testing.T) {
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/users", r.URL.Path)
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"` + uuid.NewString() + `","login":"user"}]`)), Header: make(http.Header)}, nil
	})

	users, err := client.ListUsers(context.Background())

	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "user", users[0].Login)
}

func TestUserClientListsExecutorsThroughAuthenticatedEndpoint(t *testing.T) {
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/users/executors", r.URL.Path)
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		return response(http.StatusOK, `[{"id":"`+uuid.NewString()+`","fullName":"Executor"}]`), nil
	})

	users, err := client.ListExecutors(context.Background())
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "Executor", users[0].FullName)
}

func TestUserClientCreateDoesNotRequireDesktopGeneratedPassword(t *testing.T) {
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Empty(t, body["password"])
		return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"` + uuid.NewString() + `","login":"new","temporaryPassword":"TempPassw0rd!"}`)), Header: make(http.Header)}, nil
	})

	user, err := client.CreateUser(context.Background(), models.CreateUserRequest{Login: "new"})

	require.NoError(t, err)
	assert.Equal(t, "TempPassw0rd!", user.TemporaryPassword)
}

func TestUserClientMapsForbiddenError(t *testing.T) {
	client := userClientWithToken(t, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"code":"forbidden","error":"недостаточно прав"}`)), Header: make(http.Header)}, nil
	})

	_, err := client.ListUsers(context.Background())

	assert.ErrorIs(t, err, models.ErrForbidden)
}

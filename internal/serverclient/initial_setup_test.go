package serverclient

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialSetupClientUsesUnauthenticatedServerEndpoints(t *testing.T) {
	requestNumber := 0
	client, err := New("https://server.test")
	require.NoError(t, err)
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Empty(t, r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/auth/setup-required", r.URL.Path)
			return response(http.StatusOK, `{"required":true}`), nil
		case 2:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/auth/setup", r.URL.Path)
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "Passw0rd!", body["password"])
			return response(http.StatusNoContent, ``), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	required, err := client.NeedsInitialSetup(context.Background())
	require.NoError(t, err)
	assert.True(t, required)
	require.NoError(t, client.InitialSetup(context.Background(), "Passw0rd!"))
}

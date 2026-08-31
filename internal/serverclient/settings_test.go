package serverclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsClientUsesTypedEndpoints(t *testing.T) {
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/settings", r.URL.Path)
			return response(http.StatusOK, `[{"key":"organization_name","value":"Docflow"}]`), nil
		case 2:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/settings/organization_name", r.URL.Path)
			return response(http.StatusOK, `{"key":"organization_name","value":"Docflow"}`), nil
		case 3:
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/api/v1/settings/organization_name", r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	settings, err := client.ListSettings(context.Background())
	require.NoError(t, err)
	require.Len(t, settings, 1)
	setting, err := client.GetSystemSetting(context.Background(), "organization_name")
	require.NoError(t, err)
	assert.Equal(t, "Docflow", setting.Value)
	require.NoError(t, client.UpdateSystemSetting(context.Background(), "organization_name", "New"))
}

func TestSettingsClientRejectsEmptyKeyBeforeRequest(t *testing.T) {
	requests := 0
	client := userClientWithToken(t, func(*http.Request) (*http.Response, error) {
		requests++
		return response(http.StatusInternalServerError, ""), nil
	})

	setting, err := client.GetSystemSetting(context.Background(), " ")
	assert.Nil(t, setting)
	require.Error(t, err)
	require.Error(t, client.UpdateSystemSetting(context.Background(), "", "value"))
	assert.Zero(t, requests)
}

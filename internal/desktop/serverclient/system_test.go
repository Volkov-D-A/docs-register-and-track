package serverclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemClientUsesPublicEndpoints(t *testing.T) {
	client, err := New("https://server.test")
	require.NoError(t, err)
	request := 0
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		request++
		assert.Empty(t, r.Header.Get("Authorization"))
		if request == 1 {
			assert.Equal(t, "/api/v1/system/compatibility", r.URL.Path)
			assert.Equal(t, "1.0.6", r.URL.Query().Get("clientVersion"))
			return jsonResponse(`{"compatible":true,"code":"compatible","apiVersion":"v1","serverVersion":"1.0.6"}`), nil
		}
		assert.Equal(t, "/api/v1/system/status", r.URL.Path)
		return jsonResponse(`{"status":"ready","code":"ready","apiVersion":"v1","serverVersion":"1.0.6"}`), nil
	})

	compatibility, err := client.Compatibility(context.Background(), "1.0.6")
	require.NoError(t, err)
	assert.True(t, compatibility.Compatible)
	status, err := client.SystemStatus(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ready", status.Status)
}

func TestSystemClientClassifiesTransportAndProtocolErrors(t *testing.T) {
	client, err := New("https://server.test")
	require.NoError(t, err)
	client.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	_, err = client.SystemStatus(context.Background())
	assert.Equal(t, SystemErrorUnavailable, SystemRequestErrorKind(err))

	client.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})
	_, err = client.SystemStatus(context.Background())
	assert.Equal(t, SystemErrorProtocol, SystemRequestErrorKind(err))
}

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

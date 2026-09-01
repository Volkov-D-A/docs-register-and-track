package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTechnicalLogsRequireSession(t *testing.T) {
	api, _, _ := authenticatedUserAPI(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/logs", strings.NewReader(`{"events":[{"message":"test"}]}`))
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, req)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestTechnicalLogsUseAuthenticatedIdentityAndFilterSecrets(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	payload := `{"events":[{"timestamp":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","level":"WARN","message":"desktop warning","attributes":{"component":"wails","app_user_id":"spoofed","access_token":"secret"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/logs", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, req)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	line := output.String()
	assert.Contains(t, line, `"msg":"desktop warning"`)
	assert.Contains(t, line, `"source":"desktop"`)
	assert.NotContains(t, line, "Admin")
	assert.Contains(t, line, `"desktop_component":"wails"`)
	assert.NotContains(t, line, "spoofed")
	assert.NotContains(t, line, "secret")
}

func TestTechnicalLogsRejectOversizedBatch(t *testing.T) {
	api, _, token := authenticatedUserAPI(t, nil)
	events := strings.Repeat(`{"message":"test"},`, maxTechnicalLogBatch) + `{"message":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/logs", strings.NewReader(`{"events":[`+events+`]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, req)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

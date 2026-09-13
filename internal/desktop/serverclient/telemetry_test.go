package serverclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestTelemetryClientSendsAuthenticatedBatch(t *testing.T) {
	timestamp := time.Now().UTC().Truncate(time.Second)
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/telemetry/logs", r.URL.Path)
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var batch models.TechnicalLogBatch
		require.NoError(t, json.NewDecoder(r.Body).Decode(&batch))
		require.Len(t, batch.Events, 1)
		assert.Equal(t, timestamp, batch.Events[0].Timestamp)
		assert.Equal(t, "desktop started", batch.Events[0].Message)
		assert.Equal(t, "desktop", batch.Events[0].Attributes["component"])
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})

	err := client.SendTechnicalLogs(context.Background(), []models.TechnicalLogEvent{{
		Timestamp: timestamp,
		Level:     "INFO",
		Message:   "desktop started",
		Attributes: map[string]string{
			"component": "desktop",
		},
	}})

	require.NoError(t, err)
}

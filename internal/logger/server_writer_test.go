package logger

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type recordingTelemetryClient struct {
	mu     sync.Mutex
	events []models.TechnicalLogEvent
}

func (c *recordingTelemetryClient) SendTechnicalLogs(_ context.Context, events []models.TechnicalLogEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, events...)
	return nil
}

func TestServerAsyncWriterNormalizesAndFiltersCLEF(t *testing.T) {
	client := &recordingTelemetryClient{}
	writer := NewServerAsyncWriter(client)

	payload := []byte(`{"@t":"2030-01-01T00:00:00Z","@l":"Warning","@m":"desktop warning","component":"wails","app_user_id":"spoofed","access_token":"secret","request":{"id":"42","password":"nested-secret"}}`)
	n, err := writer.Write(payload)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)
	require.NoError(t, writer.Close())

	client.mu.Lock()
	defer client.mu.Unlock()
	require.Len(t, client.events, 1)
	event := client.events[0]
	assert.Equal(t, "desktop warning", event.Message)
	assert.Equal(t, "Warning", event.Level)
	assert.Equal(t, "wails", event.Attributes["component"])
	assert.Equal(t, "42", event.Attributes["request.id"])
	assert.NotContains(t, event.Attributes, "app_user_id")
	assert.NotContains(t, event.Attributes, "access_token")
	assert.NotContains(t, event.Attributes, "request.password")
}

func TestServerAsyncWriterIgnoresMalformedCLEF(t *testing.T) {
	client := &recordingTelemetryClient{}
	writer := NewServerAsyncWriter(client)
	_, err := writer.Write([]byte(`not-json`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	assert.Empty(t, client.events)
}

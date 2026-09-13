package serverclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseErrorPreservesRequestIDAndSentinel(t *testing.T) {
	id := uuid.NewString()
	for _, code := range []string{"invalid_credentials", "session_invalid", "validation_error", "migration_apply_failed"} {
		resp := sessionResponse(409, fmt.Sprintf(`{"code":%q,"error":"safe message","requestId":%q}`, code, id))
		err := decodeAuthError(resp)
		assert.Equal(t, id, models.ErrorRequestID(fmt.Errorf("wrapped: %w", err)))
		if code == "session_invalid" {
			assert.ErrorIs(t, err, models.ErrUnauthorized)
		}
		if code == "invalid_credentials" {
			assert.ErrorIs(t, err, models.ErrInvalidCredentials)
		}
	}
	// Invalid identifiers from legacy/proxy responses are never echoed to the UI.
	resp := sessionResponse(500, `{"error":"private SQL","requestId":"private /config"}`)
	err := decodeAuthError(resp)
	assert.Empty(t, models.ErrorRequestID(err))
	assert.NotContains(t, err.Error(), "private")
}

func TestServerFailuresCannotMasqueradeAsSessionErrors(t *testing.T) {
	for _, body := range []string{`{"code":"session_invalid","error":"private SQL"}`, "private proxy text"} {
		err := decodeAuthError(sessionResponse(500, body))
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, 500, appErr.StatusCode())
		assert.False(t, errors.Is(err, models.ErrUnauthorized))
		assert.NotContains(t, err.Error(), "private")
	}
}

func TestRequestIDSurvivesAllHTTPErrorPaths(t *testing.T) {
	operations := map[string]func(*Client) error{
		"bearer": func(c *Client) error { _, err := c.Me(context.Background()); return err },
		"upload": func(c *Client) error {
			_, err := c.UploadAttachment(context.Background(), "doc", "", "a.txt", 1, strings.NewReader("a"))
			return err
		},
		"download":  func(c *Client) error { _, _, err := c.GetAttachmentContent(context.Background(), "file"); return err },
		"migration": func(c *Client) error { _, err := c.Apply(context.Background(), "admin", "wrong"); return err },
		"system":    func(c *Client) error { _, err := c.SystemStatus(context.Background()); return err },
	}
	for name, op := range operations {
		t.Run(name, func(t *testing.T) {
			c := sessionTestClient(t)
			id := uuid.NewString()
			c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 401, Header: http.Header{"X-Request-Id": []string{id}}, Body: io.NopCloser(strings.NewReader(`{"code":"session_invalid"}`))}, nil
			})
			err := op(c)
			require.Error(t, err)
			assert.Equal(t, id, models.ErrorRequestID(err))
			if name != "migration" && name != "system" {
				assert.False(t, c.SessionState().Authenticated)
			} else {
				assert.True(t, c.SessionState().Authenticated)
			}
		})
	}
}

func TestMaintenanceResponseRetainsMeaning(t *testing.T) {
	err := decodeAuthError(sessionResponse(503, `{"code":"maintenance","error":"private SQL"}`))
	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, "MAINTENANCE", appErr.SafeKind())
	assert.Equal(t, 503, appErr.StatusCode())
	assert.NotContains(t, err.Error(), "private")
}

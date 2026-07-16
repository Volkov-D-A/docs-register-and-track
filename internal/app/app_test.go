package app

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestFormatBackendErrorSerializesStructuredError(t *testing.T) {
	formatted, ok := formatBackendError(models.ErrPasswordChangeRequired).(string)
	require.True(t, ok)

	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(formatted), &payload))
	require.Equal(t, "PASSWORD_CHANGE_REQUIRED", payload.Code)
	require.Equal(t, "необходимо сменить пароль", payload.Message)
	require.Equal(t, 403, payload.Status)
}

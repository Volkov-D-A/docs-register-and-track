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

func TestBackendErrorCarriesRequestIDThroughServiceWrapper(t *testing.T) {
	id := "435a3c95-0024-4907-b2df-de1b5b5c31bc"
	err := models.NewConflictWrapped("Не удалось применить миграции", models.WithRequestID(models.NewInternal("private SQL", nil), id))
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(formatBackendError(err).(string)), &result))
	require.Equal(t, id, result["requestId"])
	require.Equal(t, "Не удалось применить миграции", result["message"])

	require.NoError(t, json.Unmarshal([]byte(formatBackendError(models.WithRequestID(models.NewInternal("private SQL", nil), id)).(string)), &result))
	require.Equal(t, id, result["requestId"])
	require.NotContains(t, result["message"], "private")
}

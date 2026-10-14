package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func requireAppError(t *testing.T, err error, kind string, code int, message string) *models.AppError {
	t.Helper()

	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, kind, appErr.Kind)
	assert.Equal(t, code, appErr.Code)
	if message != "" {
		assert.Contains(t, appErr.Message, message)
	}
	return appErr
}

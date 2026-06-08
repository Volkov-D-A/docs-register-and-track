package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAdminNumberOverride(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		result, err := buildAdminNumberOverride(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("insert shift", func(t *testing.T) {
		result, err := buildAdminNumberOverride(&AdminNumberOverrideRequest{
			Mode:   models.AdminNumberModeInsertShift,
			Number: 15,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, models.AdminNumberModeInsertShift, result.Mode)
		assert.Equal(t, 15, result.Number)
		assert.Empty(t, result.Suffix)
	})

	t.Run("literal with suffix", func(t *testing.T) {
		result, err := buildAdminNumberOverride(&AdminNumberOverrideRequest{
			Mode:   models.AdminNumberModeLiteral,
			Number: 15,
			Suffix: "А",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, models.AdminNumberModeLiteral, result.Mode)
		assert.Equal(t, 15, result.Number)
		assert.Equal(t, "А", result.Suffix)
	})

	t.Run("rejects invalid mode", func(t *testing.T) {
		result, err := buildAdminNumberOverride(&AdminNumberOverrideRequest{
			Mode:   "bad",
			Number: 15,
		})
		require.Error(t, err)
		require.Nil(t, result)
		assert.Contains(t, err.Error(), "неверный режим административной нумерации")
	})

	t.Run("rejects suffix for shift mode", func(t *testing.T) {
		result, err := buildAdminNumberOverride(&AdminNumberOverrideRequest{
			Mode:   models.AdminNumberModeInsertShift,
			Number: 15,
			Suffix: "А",
		})
		require.Error(t, err)
		require.Nil(t, result)
		assert.Contains(t, err.Error(), "для вставки со сдвигом")
	})
}

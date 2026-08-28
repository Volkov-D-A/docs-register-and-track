package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePageCounts(t *testing.T) {
	t.Run("accepts document and attachment counts", func(t *testing.T) {
		require.NoError(t, validatePageCounts(1, 0))
		require.NoError(t, validatePageCounts(12, 7))
	})

	t.Run("rejects missing document pages", func(t *testing.T) {
		err := validatePageCounts(0, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "количество листов")
	})

	t.Run("rejects negative attachment pages", func(t *testing.T) {
		err := validatePageCounts(1, -1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "листов приложения")
	})
}

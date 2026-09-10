package backup

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestScheduleTimezoneAndMissedOccurrence(t *testing.T) {
	settings := DefaultSettings()
	settings.Enabled = true
	settings.SMB = models.SMBDestination{Host: "nas", Share: "backups", User: "backup"}
	require.NoError(t, settings.Validate())
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	due, ok := settings.Due(now)
	require.True(t, ok)
	require.Equal(t, "2026-09-09T21:00:00Z", due.UTC().Format(time.RFC3339))
	settings.Weekdays = []int{int(time.Monday)}
	due, ok = settings.Due(now)
	require.True(t, ok)
	require.Equal(t, time.Monday, due.Weekday())
	settings.Enabled = false
	_, ok = settings.Due(now)
	require.False(t, ok)
	settings.Timezone = "invalid"
	require.Error(t, settings.Validate())
}

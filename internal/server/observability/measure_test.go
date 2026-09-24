package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMeasurePreservesResultsAndRecordsOutcomes(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		var metrics *Registry
		if enabled {
			metrics = NewRegistry(8)
		}
		calls := 0
		value, err := Measure(metrics, "documents.register", func() (int, error) { calls++; return 42, context.DeadlineExceeded })
		require.Equal(t, 42, value)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		err = MeasureError(metrics, "documents.register", func() error { calls++; return nil })
		require.NoError(t, err)
		err = MeasureError(metrics, "documents.register", func() error { calls++; return context.Canceled })
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 3, calls)
		if enabled {
			snapshots := metrics.Snapshot()
			require.Len(t, snapshots, 1)
			require.Equal(t, "documents.register", snapshots[0].Name)
			require.EqualValues(t, 3, snapshots[0].Count)
			require.EqualValues(t, 2, snapshots[0].Errors)
			require.EqualValues(t, 1, snapshots[0].DeadlineExceeded)
		}
	}
}

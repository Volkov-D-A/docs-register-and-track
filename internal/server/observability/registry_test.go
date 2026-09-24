package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrySnapshotCalculatesCountersAndPercentiles(t *testing.T) {
	registry := NewRegistry(100)
	for i := 1; i <= 100; i++ {
		var err error
		switch i {
		case 10:
			err = context.DeadlineExceeded
		case 20:
			err = errors.New("failed")
		}
		registry.Observe("documents.get_list", time.Duration(i)*time.Millisecond, err)
	}

	metrics := registry.Snapshot()
	require.Len(t, metrics, 1)
	assert.Equal(t, "documents.get_list", metrics[0].Name)
	assert.EqualValues(t, 100, metrics[0].Count)
	assert.EqualValues(t, 2, metrics[0].Errors)
	assert.EqualValues(t, 1, metrics[0].DeadlineExceeded)
	assert.Equal(t, 50*time.Millisecond, metrics[0].P50)
	assert.Equal(t, 95*time.Millisecond, metrics[0].P95)
	assert.Equal(t, 99*time.Millisecond, metrics[0].P99)
}

func TestRegistryRetainsBoundedLatencyWindow(t *testing.T) {
	registry := NewRegistry(2)
	registry.Observe("operation", time.Millisecond, nil)
	registry.Observe("operation", 2*time.Millisecond, nil)
	registry.Observe("operation", 10*time.Millisecond, nil)

	metric := registry.Snapshot()[0]
	assert.EqualValues(t, 3, metric.Count)
	assert.Equal(t, 2*time.Millisecond, metric.P50)
	assert.Equal(t, 10*time.Millisecond, metric.P95)
}

func TestRegistryHandlesNilAndEmptyNames(t *testing.T) {
	var nilRegistry *Registry
	nilRegistry.Observe("ignored", time.Millisecond, nil)
	assert.Nil(t, nilRegistry.Snapshot())

	registry := NewRegistry(1)
	registry.Observe("", time.Millisecond, nil)
	assert.Empty(t, registry.Snapshot())
}

func TestRegistryStoresLatestGaugeValue(t *testing.T) {
	registry := NewRegistry(1)
	registry.SetGauge("database.pool.in_use", 2)
	registry.SetGauge("database.pool.in_use", 1)
	registry.SetGauge("database.pool.idle", 3)

	assert.Equal(t, []GaugeSnapshot{
		{Name: "database.pool.idle", Value: 3},
		{Name: "database.pool.in_use", Value: 1},
	}, registry.Gauges())
}

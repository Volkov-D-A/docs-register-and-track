package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMigrationStatusReader struct {
	mu     sync.RWMutex
	status database.MigrationStatus
	err    error
}

func (r *fakeMigrationStatusReader) GetMigrationStatus(path string) (*database.MigrationStatus, error) {
	if path != database.DefaultMigrationsPath {
		return nil, errors.New("unexpected migration path")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.err != nil {
		return nil, r.err
	}
	status := r.status
	return &status, nil
}

func (r *fakeMigrationStatusReader) set(status database.MigrationStatus, err error) {
	r.mu.Lock()
	r.status = status
	r.err = err
	r.mu.Unlock()
}

type blockingBackgroundWorker struct {
	starts atomic.Int32
	stops  atomic.Int32
}

func (w *blockingBackgroundWorker) Run(ctx context.Context) {
	w.starts.Add(1)
	<-ctx.Done()
	w.stops.Add(1)
}

func readyMigrationStatus() database.MigrationStatus {
	return database.MigrationStatus{
		CurrentVersion:         10,
		AvailableCount:         10,
		LatestAvailableVersion: 10,
		UpToDate:               true,
		Compatible:             true,
	}
}

func stopLifecycle(t *testing.T, lifecycle *backgroundLifecycle) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, lifecycle.Stop(ctx))
}

func TestBackgroundLifecycleWaitsForContextAndReadySchema(t *testing.T) {
	reader := &fakeMigrationStatusReader{status: readyMigrationStatus()}
	worker := &blockingBackgroundWorker{}
	lifecycle := newBackgroundLifecycle(reader, worker, nil)

	lifecycle.ReconcileSchema()
	assert.Zero(t, worker.starts.Load())
	require.NoError(t, lifecycle.CheckReady())

	reader.set(database.MigrationStatus{
		CurrentVersion:         9,
		AvailableCount:         10,
		LatestAvailableVersion: 10,
		Compatible:             true,
	}, nil)
	lifecycle.SetApplicationContext(context.Background())
	lifecycle.ReconcileSchema()
	assert.Zero(t, worker.starts.Load())
	require.Error(t, lifecycle.CheckReady())

	reader.set(readyMigrationStatus(), nil)
	lifecycle.ReconcileSchema()
	require.Eventually(t, func() bool { return worker.starts.Load() == 1 }, time.Second, time.Millisecond)
	require.NoError(t, lifecycle.CheckReady())
	stopLifecycle(t, lifecycle)
}

func TestBackgroundLifecycleStartsWorkerOnlyOnce(t *testing.T) {
	reader := &fakeMigrationStatusReader{status: readyMigrationStatus()}
	worker := &blockingBackgroundWorker{}
	var startupRuns atomic.Int32
	lifecycle := newBackgroundLifecycle(reader, worker, func(context.Context) error {
		startupRuns.Add(1)
		return nil
	})
	lifecycle.SetApplicationContext(context.Background())

	var callers sync.WaitGroup
	for range 20 {
		callers.Add(1)
		go func() {
			defer callers.Done()
			lifecycle.ReconcileSchema()
		}()
	}
	callers.Wait()

	require.Eventually(t, func() bool {
		return worker.starts.Load() == 1 && startupRuns.Load() == 1
	}, time.Second, time.Millisecond)
	stopLifecycle(t, lifecycle)
	assert.Equal(t, int32(1), worker.stops.Load())
}

func TestBackgroundLifecycleBlocksOnUnavailableOrUnsafeSchema(t *testing.T) {
	tests := []struct {
		name   string
		status database.MigrationStatus
		err    error
	}{
		{name: "status error", err: assert.AnError},
		{name: "dirty", status: database.MigrationStatus{CurrentVersion: 10, LatestAvailableVersion: 10, Dirty: true}},
		{name: "too new", status: database.MigrationStatus{CurrentVersion: 11, LatestAvailableVersion: 10, SchemaTooNew: true}},
		{name: "outdated", status: database.MigrationStatus{CurrentVersion: 9, LatestAvailableVersion: 10, Compatible: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeMigrationStatusReader{status: tt.status, err: tt.err}
			worker := &blockingBackgroundWorker{}
			lifecycle := newBackgroundLifecycle(reader, worker, nil)
			lifecycle.SetApplicationContext(context.Background())

			lifecycle.ReconcileSchema()

			assert.Zero(t, worker.starts.Load())
			require.Error(t, lifecycle.CheckReady())
		})
	}
}

func TestBackgroundLifecycleRollbackStopsAndCanRecover(t *testing.T) {
	reader := &fakeMigrationStatusReader{status: readyMigrationStatus()}
	worker := &blockingBackgroundWorker{}
	lifecycle := newBackgroundLifecycle(reader, worker, nil)
	lifecycle.SetApplicationContext(context.Background())
	lifecycle.ReconcileSchema()
	require.Eventually(t, func() bool { return worker.starts.Load() == 1 }, time.Second, time.Millisecond)

	require.NoError(t, lifecycle.PrepareRollback())
	require.Eventually(t, func() bool { return worker.stops.Load() == 1 }, time.Second, time.Millisecond)
	require.Error(t, lifecycle.CheckReady())

	lifecycle.CompleteRollback(false)
	require.Eventually(t, func() bool { return worker.starts.Load() == 2 }, time.Second, time.Millisecond)
	require.NoError(t, lifecycle.CheckReady())
	stopLifecycle(t, lifecycle)
}

func TestBackgroundLifecycleSuccessfulRollbackStaysInMaintenance(t *testing.T) {
	reader := &fakeMigrationStatusReader{status: readyMigrationStatus()}
	worker := &blockingBackgroundWorker{}
	lifecycle := newBackgroundLifecycle(reader, worker, nil)
	lifecycle.SetApplicationContext(context.Background())
	lifecycle.ReconcileSchema()
	require.Eventually(t, func() bool { return worker.starts.Load() == 1 }, time.Second, time.Millisecond)

	require.NoError(t, lifecycle.PrepareRollback())
	lifecycle.CompleteRollback(true)
	require.Error(t, lifecycle.CheckReady())
	assert.Equal(t, int32(1), worker.starts.Load())

	reader.set(readyMigrationStatus(), nil)
	lifecycle.ReconcileSchema()
	require.Eventually(t, func() bool { return worker.starts.Load() == 2 }, time.Second, time.Millisecond)
	require.NoError(t, lifecycle.CheckReady())
	stopLifecycle(t, lifecycle)
}

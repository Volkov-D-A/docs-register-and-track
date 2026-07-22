package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const backgroundStopTimeout = 15 * time.Second

type migrationStatusReader interface {
	GetMigrationStatus(string) (*database.MigrationStatus, error)
}

type backgroundWorker interface {
	Run(context.Context)
}

type backgroundLifecycleState uint8

const (
	backgroundStopped backgroundLifecycleState = iota
	backgroundRunning
	backgroundStopping
)

// backgroundLifecycle owns schema-dependent workers. It deliberately lives in
// the composition layer: services report schema changes without depending on
// concrete outbox or attachment implementations.
type backgroundLifecycle struct {
	statusReader migrationStatusReader
	worker       backgroundWorker
	startupWork  func(context.Context) error

	reconcileMu  sync.Mutex
	mu           sync.Mutex
	appContext   context.Context
	workerCancel context.CancelFunc
	workerDone   chan struct{}
	state        backgroundLifecycleState
	maintenance  bool
	rollback     bool
}

func newBackgroundLifecycle(
	statusReader migrationStatusReader,
	worker backgroundWorker,
	startupWork func(context.Context) error,
) *backgroundLifecycle {
	return &backgroundLifecycle{
		statusReader: statusReader,
		worker:       worker,
		startupWork:  startupWork,
		state:        backgroundStopped,
		maintenance:  true,
	}
}

func (l *backgroundLifecycle) SetApplicationContext(ctx context.Context) {
	l.mu.Lock()
	l.appContext = ctx
	l.mu.Unlock()
}

func (l *backgroundLifecycle) ReconcileSchema() {
	l.reconcileMu.Lock()
	defer l.reconcileMu.Unlock()
	l.reconcileSchemaLocked()
}

func (l *backgroundLifecycle) reconcileSchemaLocked() {
	status, err := l.statusReader.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil || status == nil {
		l.setMaintenance(true)
		l.stopWithTimeout("migration status is unavailable")
		slog.Warn("background services were not started because migration status is unavailable", "error", err)
		return
	}

	if !status.UpToDate || !status.Compatible {
		l.setMaintenance(true)
		l.stopWithTimeout("database schema is not ready")
		slog.Info(
			"background services are deferred until migrations are applied",
			"current_version", status.CurrentVersion,
			"required_version", status.LatestAvailableVersion,
			"dirty", status.Dirty,
			"schema_too_new", status.SchemaTooNew,
		)
		return
	}

	l.mu.Lock()
	if l.rollback {
		l.mu.Unlock()
		return
	}
	l.maintenance = false
	if l.appContext == nil || l.state != backgroundStopped {
		l.mu.Unlock()
		return
	}

	workerContext, cancel := context.WithCancel(l.appContext)
	done := make(chan struct{})
	l.workerCancel = cancel
	l.workerDone = done
	l.state = backgroundRunning
	l.mu.Unlock()

	go l.runWorker(workerContext, done)
	if l.startupWork != nil {
		go func() {
			if err := l.startupWork(workerContext); err != nil && workerContext.Err() == nil {
				slog.Warn("schema-dependent startup work failed", "error", err)
			}
		}()
	}
}

func (l *backgroundLifecycle) runWorker(ctx context.Context, done chan struct{}) {
	defer close(done)
	l.worker.Run(ctx)

	l.mu.Lock()
	if l.workerDone == done {
		l.state = backgroundStopped
		l.workerCancel = nil
		l.workerDone = nil
	}
	l.mu.Unlock()
}

func (l *backgroundLifecycle) PrepareRollback() error {
	l.reconcileMu.Lock()
	defer l.reconcileMu.Unlock()

	l.mu.Lock()
	l.maintenance = true
	l.rollback = true
	l.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), backgroundStopTimeout)
	defer cancel()
	if err := l.Stop(ctx); err != nil {
		l.mu.Lock()
		l.rollback = false
		l.mu.Unlock()
		l.reconcileSchemaLocked()
		return err
	}
	return nil
}

func (l *backgroundLifecycle) CompleteRollback(success bool) {
	l.reconcileMu.Lock()
	defer l.reconcileMu.Unlock()

	l.mu.Lock()
	l.rollback = false
	l.mu.Unlock()
	if success {
		return
	}
	l.reconcileSchemaLocked()
}

func (l *backgroundLifecycle) CheckReady() error {
	l.mu.Lock()
	maintenance := l.maintenance
	l.mu.Unlock()
	if !maintenance {
		return nil
	}
	return models.NewConflict("Схема базы данных требует обновления. Обычная работа заблокирована до успешного применения миграций.")
}

func (l *backgroundLifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	if l.state == backgroundStopped {
		l.mu.Unlock()
		return nil
	}
	if l.state == backgroundRunning {
		l.state = backgroundStopping
		l.workerCancel()
	}
	done := l.workerDone
	l.mu.Unlock()

	select {
	case <-done:
		l.mu.Lock()
		if l.workerDone == done {
			l.state = backgroundStopped
			l.workerCancel = nil
			l.workerDone = nil
		}
		l.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *backgroundLifecycle) setMaintenance(maintenance bool) {
	l.mu.Lock()
	l.maintenance = maintenance
	l.mu.Unlock()
}

func (l *backgroundLifecycle) stopWithTimeout(reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), backgroundStopTimeout)
	defer cancel()
	if err := l.Stop(ctx); err != nil {
		slog.Warn("background services did not stop before timeout", "reason", reason, "error", err)
	}
}

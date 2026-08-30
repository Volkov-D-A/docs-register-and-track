package background

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const stopTimeout = 15 * time.Second

type MigrationStatusReader interface {
	GetMigrationStatus(string) (*database.MigrationStatus, error)
}

type Worker interface {
	Run(context.Context)
}

type lifecycleState uint8

const (
	stopped lifecycleState = iota
	running
	stopping
)

// Lifecycle owns background workers that may run only against a compatible,
// fully migrated schema. A nil worker keeps the schema maintenance gate while
// deliberately disabling in-process background consumption.
type Lifecycle struct {
	statusReader MigrationStatusReader
	worker       Worker
	startupWork  func(context.Context) error

	reconcileMu  sync.Mutex
	mu           sync.Mutex
	appContext   context.Context
	workerCancel context.CancelFunc
	workerDone   chan struct{}
	state        lifecycleState
	maintenance  bool
	rollback     bool
}

func NewLifecycle(statusReader MigrationStatusReader, worker Worker, startupWork func(context.Context) error) *Lifecycle {
	return &Lifecycle{
		statusReader: statusReader,
		worker:       worker,
		startupWork:  startupWork,
		state:        stopped,
		maintenance:  true,
	}
}

func (l *Lifecycle) SetApplicationContext(ctx context.Context) {
	l.mu.Lock()
	l.appContext = ctx
	l.mu.Unlock()
}

func (l *Lifecycle) ReconcileSchema() {
	l.reconcileMu.Lock()
	defer l.reconcileMu.Unlock()
	l.reconcileSchemaLocked()
}

func (l *Lifecycle) reconcileSchemaLocked() {
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
	if l.worker == nil || l.appContext == nil || l.state != stopped {
		l.mu.Unlock()
		return
	}

	workerContext, cancel := context.WithCancel(l.appContext)
	done := make(chan struct{})
	l.workerCancel = cancel
	l.workerDone = done
	l.state = running
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

func (l *Lifecycle) runWorker(ctx context.Context, done chan struct{}) {
	defer close(done)
	l.worker.Run(ctx)

	l.mu.Lock()
	if l.workerDone == done {
		l.state = stopped
		l.workerCancel = nil
		l.workerDone = nil
	}
	l.mu.Unlock()
}

func (l *Lifecycle) PrepareRollback() error {
	l.reconcileMu.Lock()
	defer l.reconcileMu.Unlock()

	l.mu.Lock()
	l.maintenance = true
	l.rollback = true
	l.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
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

func (l *Lifecycle) CompleteRollback(success bool) {
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

func (l *Lifecycle) CheckReady() error {
	l.mu.Lock()
	maintenance := l.maintenance
	l.mu.Unlock()
	if !maintenance {
		return nil
	}
	return models.NewConflict("Схема базы данных требует обновления. Обычная работа заблокирована до успешного применения миграций.")
}

func (l *Lifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	if l.state == stopped {
		l.mu.Unlock()
		return nil
	}
	if l.state == running {
		l.state = stopping
		l.workerCancel()
	}
	done := l.workerDone
	l.mu.Unlock()

	select {
	case <-done:
		l.mu.Lock()
		if l.workerDone == done {
			l.state = stopped
			l.workerCancel = nil
			l.workerDone = nil
		}
		l.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *Lifecycle) setMaintenance(maintenance bool) {
	l.mu.Lock()
	l.maintenance = maintenance
	l.mu.Unlock()
}

func (l *Lifecycle) stopWithTimeout(reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	if err := l.Stop(ctx); err != nil {
		slog.Warn("background services did not stop before timeout", "reason", reason, "error", err)
	}
}

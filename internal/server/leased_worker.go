package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
)

type contextWorker interface {
	Run(context.Context)
}

// leasedWorker holds the schema/worker advisory lease only while the worker is
// actually running. The management API can therefore stop the worker, wait for
// lease release, migrate, and start it again in the same server process.
type leasedWorker struct {
	db     *database.DB
	worker contextWorker
}

func (w *leasedWorker) Run(ctx context.Context) {
	lease, acquired := w.waitForLease(ctx)
	if !acquired {
		return
	}
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()
		if err := lease.Release(releaseCtx); err != nil {
			slog.Error("failed to release outbox worker lease", "error", err)
		}
	}()
	w.worker.Run(ctx)
}

func (w *leasedWorker) waitForLease(ctx context.Context) (*database.BackgroundWorkerLease, bool) {
	waitingLogged := false
	for {
		leaseCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		lease, acquired, err := w.db.TryAcquireBackgroundWorkerLease(leaseCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.Error("failed to acquire outbox worker lease", "error", err)
		}
		if acquired {
			if waitingLogged {
				slog.Info("outbox worker lease acquired after waiting")
			}
			return lease, true
		}
		if ctx.Err() != nil {
			return nil, false
		}
		if !waitingLogged {
			slog.Warn("outbox worker is waiting for the singleton lease")
			waitingLogged = true
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, false
		case <-timer.C:
		}
	}
}

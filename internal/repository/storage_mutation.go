package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/coordination"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
)

const (
	storageMutationLeaseSeconds  = int64(90)
	storageMutationRenewInterval = 30 * time.Second
)

type storageMutation struct {
	db     *database.DB
	token  uuid.UUID
	ctx    context.Context
	cancel context.CancelFunc
	stop   chan struct{}
	done   chan struct{}

	finishOnce sync.Once
	finishErr  error
	errMu      sync.Mutex
	renewErr   error
}

var _ coordination.StorageMutation = (*storageMutation)(nil)

// BeginStorageMutation registers the operation before MinIO can change. The
// registration invalidates an overlapping bucket scan and is renewed until
// Finish is called.
func (r *AttachmentRepository) BeginStorageMutation(parent context.Context) (coordination.StorageMutation, error) {
	if parent == nil {
		parent = context.Background()
	}
	token := uuid.New()
	tx, err := r.db.BeginTx(parent, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var singleton bool
	if err := tx.QueryRow(`SELECT id FROM storage_statistics WHERE id = true FOR UPDATE`).Scan(&singleton); err != nil {
		return nil, fmt.Errorf("failed to lock storage statistics: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM storage_statistics_mutations WHERE lease_until < CURRENT_TIMESTAMP`); err != nil {
		return nil, fmt.Errorf("failed to reap stale storage mutations: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO storage_statistics_mutations (token, lease_until)
		VALUES ($1, CURRENT_TIMESTAMP + ($2 * INTERVAL '1 second'))
	`, token, storageMutationLeaseSeconds); err != nil {
		return nil, fmt.Errorf("failed to register storage mutation: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE storage_statistics
		SET mutation_revision = mutation_revision + 1,
			refresh_token = NULL,
			refresh_lease_until = NULL,
			refresh_revision = NULL
		WHERE id = true
	`); err != nil {
		return nil, fmt.Errorf("failed to invalidate storage statistics refresh: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit storage mutation registration: %w", err)
	}

	ctx, cancel := context.WithCancel(parent)
	mutation := &storageMutation{
		db: r.db, token: token, ctx: ctx, cancel: cancel,
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	go mutation.renewLease()
	return mutation, nil
}

func (m *storageMutation) Context() context.Context { return m.ctx }

func (m *storageMutation) renewLease() {
	defer close(m.done)
	ticker := time.NewTicker(storageMutationRenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			result, err := m.db.ExecContext(m.ctx, `
				UPDATE storage_statistics_mutations
				SET lease_until = CURRENT_TIMESTAMP + ($2 * INTERVAL '1 second')
				WHERE token = $1
			`, m.token, storageMutationLeaseSeconds)
			if err == nil {
				var affected int64
				affected, err = result.RowsAffected()
				if err == nil && affected != 1 {
					err = sql.ErrNoRows
				}
			}
			if err != nil {
				m.errMu.Lock()
				m.renewErr = fmt.Errorf("failed to renew storage mutation lease: %w", err)
				m.errMu.Unlock()
				m.cancel()
				return
			}
		}
	}
}

func (m *storageMutation) Finish() error {
	m.finishOnce.Do(func() {
		close(m.stop)
		<-m.done
		m.cancel()

		m.errMu.Lock()
		renewErr := m.renewErr
		m.errMu.Unlock()
		_, deleteErr := m.db.ExecContext(context.Background(), `DELETE FROM storage_statistics_mutations WHERE token = $1`, m.token)
		if deleteErr != nil {
			m.finishErr = fmt.Errorf("failed to finish storage mutation: %w", deleteErr)
		} else {
			m.finishErr = renewErr
		}
	})
	return m.finishErr
}

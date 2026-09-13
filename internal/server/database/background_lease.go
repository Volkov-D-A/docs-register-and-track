package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// A stable application-specific PostgreSQL advisory lock prevents multiple
// server workers and server-owned schema changes from overlapping.
const backgroundWorkerLeaseID int64 = 0x444f43464c4f5752

type BackgroundWorkerLease struct {
	conn      *sql.Conn
	releaseMu sync.Mutex
	released  bool
}

func (db *DB) TryAcquireBackgroundWorkerLease(ctx context.Context) (*BackgroundWorkerLease, bool, error) {
	if db == nil || db.DB == nil {
		return nil, false, fmt.Errorf("database is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("reserve connection for background worker lease: %w", err)
	}
	var acquired bool
	if err := conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, backgroundWorkerLeaseID).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false, fmt.Errorf("acquire background worker lease: %w", err)
	}
	if !acquired {
		_ = conn.Close()
		return nil, false, nil
	}
	return &BackgroundWorkerLease{conn: conn}, true, nil
}

func (l *BackgroundWorkerLease) Release(ctx context.Context) error {
	if l == nil {
		return nil
	}
	l.releaseMu.Lock()
	defer l.releaseMu.Unlock()
	if l.released {
		return nil
	}
	l.released = true
	if ctx == nil {
		ctx = context.Background()
	}
	var unlocked bool
	err := l.conn.QueryRowContext(ctx, `SELECT pg_advisory_unlock($1)`, backgroundWorkerLeaseID).Scan(&unlocked)
	closeErr := l.conn.Close()
	if err != nil {
		return fmt.Errorf("release background worker lease: %w", err)
	}
	if !unlocked {
		return fmt.Errorf("background worker lease was not held by its connection")
	}
	if closeErr != nil {
		return fmt.Errorf("close background worker lease connection: %w", closeErr)
	}
	return nil
}

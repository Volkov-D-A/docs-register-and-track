package server

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/backup"
	"time"
)

// All ordinary servers and recovery commands use this lifetime lease. A second
// process must not accept writes while another process is taking a snapshot.
const instanceLeaseID int64 = backup.InstanceLeaseID

func acquireInstance(ctx context.Context, db *sql.DB) (*sql.Conn, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	var acquired bool
	if err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", instanceLeaseID).Scan(&acquired); err != nil || !acquired {
		conn.Close()
		return nil, fmt.Errorf("another server or recovery process owns this database")
	}
	return conn, nil
}
func releaseInstance(conn *sql.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", instanceLeaseID)
	conn.Close()
}

package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const InstanceLeaseID int64 = 0x444f43464c4f5702

// Restore requires an empty target and leaves a persistent marker on any failure.
// Operators must preserve that marker until a successful recovery is verified.
func Restore(ctx context.Context, pg PostgreSQL, archive, staging string, maxBytes int64, schema int) error {
	if staging == "" || !filepath.IsAbs(staging) {
		return fmt.Errorf("persistent recovery directory is required")
	}
	if err := os.MkdirAll(staging, 0700); err != nil {
		return err
	}
	work, err := os.MkdirTemp(staging, "restore-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	prepared, err := PrepareRestore(ctx, pg, archive, filepath.Join(work, "contents"), maxBytes, schema)
	if err != nil {
		return err
	}
	db, err := pg.Open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var acquired bool
	if err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", InstanceLeaseID).Scan(&acquired); err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("stop the ordinary server before restoring")
	}
	defer func() {
		limited, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn.ExecContext(limited, "SELECT pg_advisory_unlock($1)", InstanceLeaseID)
	}()
	return RestorePrepared(ctx, pg, prepared, staging, schema, true)
}

// RestorePrepared is for a coordinator already holding the server lifetime
// lease. It still requires an empty database.
func RestorePrepared(ctx context.Context, pg PostgreSQL, prepared PreparedRestore, staging string, schema int, finalize bool) error {
	m := prepared.Manifest
	dump := filepath.Join(prepared.Directory, "database.dump")
	db, err := pg.Open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = pg.RequireEmpty(ctx); err != nil {
		return err
	}
	marker := filepath.Join(staging, "recovery-required")
	if finalize {
		if err = durableMarker(marker, []byte(m.ID+"\n")); err != nil {
			return err
		}
	} else if _, err = os.Stat(marker); err != nil {
		return fmt.Errorf("replacement marker is required: %w", err)
	}
	if err = pg.Restore(ctx, dump); err != nil {
		return err
	}
	var restoredSchema int
	var dirty bool
	if err = db.QueryRowContext(ctx, "SELECT version,dirty FROM schema_migrations").Scan(&restoredSchema, &dirty); err != nil {
		return err
	}
	if dirty || restoredSchema != schema || m.Schema != restoredSchema {
		return fmt.Errorf("restored schema is incompatible or dirty")
	}
	if err = ValidateControlReferences(ctx, db); err != nil {
		return err
	}
	if !finalize {
		return nil
	}
	if _, err = db.ExecContext(ctx, "UPDATE server_sessions SET revoked_at=now() WHERE revoked_at IS NULL"); err != nil {
		return err
	}
	if _, err = db.ExecContext(ctx, `UPDATE backup_settings SET settings=jsonb_set(jsonb_set(jsonb_set(settings,'{settings,enabled}','false'),'{settings,passwordSet}','false'),'{secret}','{}')`); err != nil {
		return err
	}
	if err = writeJSONFile(filepath.Join(staging, "restore-"+m.ID+"-"+fmt.Sprint(time.Now().UnixNano())+".json"), map[string]any{"id": m.ID, "completedAt": time.Now().UTC()}); err != nil {
		return err
	}
	if err = os.Remove(marker); err != nil {
		return err
	}
	return syncDirectory(staging)
}

func durableMarker(name string, data []byte) error {
	file, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	return syncDirectory(filepath.Dir(name))
}
func syncDirectory(name string) error {
	directory, err := os.Open(name)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

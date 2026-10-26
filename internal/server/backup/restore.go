package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// restorePrepared requires the replacement coordinator to hold the server lease
// and keep the recovery marker until replacement or rollback completes.
func restorePrepared(ctx context.Context, pg PostgreSQL, prepared PreparedRestore, staging string, schema int) error {
	m := prepared.Manifest
	dump := filepath.Join(prepared.Directory, "database.dump")
	db, err := pg.Open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	marker := filepath.Join(staging, "recovery-required")
	if _, err = os.Stat(marker); err != nil {
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
	return nil
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

package backup

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	_ "github.com/lib/pq"
)

// PostgreSQL uses version-matched standard tools, never a shell or Docker API.
type PostgreSQL struct{ Config config.DatabaseConfig }

func (p PostgreSQL) command(ctx context.Context, tool string, args ...string) (*exec.Cmd, func(), error) {
	dir, err := os.MkdirTemp("", "docflow-pgpass-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	escape := func(s string) string { return strings.NewReplacer("\\", "\\\\", ":", "\\:").Replace(s) }
	for _, value := range []string{p.Config.Host, p.Config.User, p.Config.DBName, p.Config.Password} {
		if strings.ContainsAny(value, "\r\n\x00") {
			cleanup()
			return nil, nil, fmt.Errorf("unsupported PostgreSQL credential character")
		}
	}
	pass := strings.Join([]string{escape(p.Config.Host), strconv.Itoa(p.Config.Port), escape(p.Config.DBName), escape(p.Config.User), escape(p.Config.Password)}, ":") + "\n"
	file := filepath.Join(dir, "pgpass")
	if err = os.WriteFile(file, []byte(pass), 0600); err != nil {
		cleanup()
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, tool, args...)
	for _, env := range os.Environ() {
		name, _, _ := strings.Cut(env, "=")
		if !strings.HasPrefix(name, "PG") {
			cmd.Env = append(cmd.Env, env)
		}
	}
	cmd.Env = append(cmd.Env, "PGHOST="+p.Config.Host, "PGPORT="+strconv.Itoa(p.Config.Port), "PGDATABASE="+p.Config.DBName, "PGUSER="+p.Config.User, "PGSSLMODE="+p.Config.SSLMode, "PGPASSFILE="+file, "PGCONNECT_TIMEOUT=10")
	return cmd, cleanup, nil
}
func (p PostgreSQL) Open(ctx context.Context) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10", quotePG(p.Config.Host), p.Config.Port, quotePG(p.Config.User), quotePG(p.Config.Password), quotePG(p.Config.DBName), quotePG(p.Config.SSLMode))
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func quotePG(s string) string {
	return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(s) + "'"
}
func (p PostgreSQL) Dump(ctx context.Context, file string, maxBytes int64) error {
	db, err := p.Open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	var version int
	if err = db.QueryRowContext(ctx, "SHOW server_version_num").Scan(&version); err != nil {
		return err
	}
	cmd, cleanup, err := p.command(ctx, "pg_dump", "--version")
	if err != nil {
		return err
	}
	defer cleanup()
	raw, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("pg_dump unavailable: %w", err)
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 3 {
		return fmt.Errorf("invalid pg_dump version")
	}
	major, err := strconv.Atoi(strings.Split(fields[2], ".")[0])
	if err != nil || major != version/10000 {
		return fmt.Errorf("pg_dump major version must match PostgreSQL")
	}
	cmd, cleanupDump, err := p.command(ctx, "pg_dump", "--format=custom", "--no-owner", "--no-acl")
	if err != nil {
		return err
	}
	defer cleanupDump()
	destination, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	cmd.Stdout = &boundedWriter{out: destination, remaining: maxBytes}
	runErr := cmd.Run()
	syncErr := destination.Sync()
	closeErr := destination.Close()
	if runErr != nil {
		return fmt.Errorf("pg_dump failed or exceeded staging limit: %w", runErr)
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	f, err := os.OpenFile(file, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func (p PostgreSQL) ValidateDump(ctx context.Context, file string) error {
	cmd, cleanup, err := p.command(ctx, "pg_restore", "--list", file)
	if err != nil {
		return err
	}
	defer cleanup()
	cmd.Stdout = io.Discard
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("invalid PostgreSQL dump: %w", err)
	}
	return nil
}
func (p PostgreSQL) RequireEmpty(ctx context.Context) error {
	db, err := p.Open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	var count int
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%' AND c.relkind IN ('r','p','v','m','S','f')`).Scan(&count)
	if err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("restore requires an empty database")
	}
	return nil
}
func (p PostgreSQL) Restore(ctx context.Context, file string) error {
	if err := p.RequireEmpty(ctx); err != nil {
		return err
	}
	cmd, cleanup, err := p.command(ctx, "pg_restore", "--exit-on-error", "--single-transaction", "--no-owner", "--no-acl", "--dbname="+p.Config.DBName, file)
	if err != nil {
		return err
	}
	defer cleanup()
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore failed: %w", err)
	}
	return nil
}
func ValidateReferences(ctx context.Context, db *sql.DB, m Manifest) error {
	objects := map[string]int64{}
	for _, obj := range m.Objects {
		objects[obj.Key] = obj.Size
	}
	rows, err := db.QueryContext(ctx, "SELECT storage_path,file_size FROM attachments WHERE deletion_requested_at IS NULL")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var size int64
		if err = rows.Scan(&key, &size); err != nil {
			return err
		}
		actual, ok := objects[key]
		if !ok || actual != size {
			return fmt.Errorf("active attachment is missing or has a different size: %s", key)
		}
	}
	return rows.Err()
}

type boundedWriter struct {
	out       io.Writer
	remaining int64
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, fmt.Errorf("staging limit exceeded")
	}
	n, err := w.out.Write(p)
	w.remaining -= int64(n)
	return n, err
}

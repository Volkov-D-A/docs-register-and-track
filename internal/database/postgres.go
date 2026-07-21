package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
)

const (
	defaultMaxOpenConns     = 5
	defaultMaxIdleConns     = 2
	defaultConnMaxIdleTime  = 5 * time.Minute
	defaultConnMaxLifetime  = 30 * time.Minute
	defaultStartupTimeout   = 10 * time.Second
	defaultOperationTimeout = 30 * time.Second
)

// DB представляет собой обертку над подключением к базе данных SQL.
type DB struct {
	*sql.DB
	operationTimeout time.Duration
	metrics          *observability.Registry
	poolMu           sync.Mutex
	lastPoolStats    sql.DBStats
}

// MigrationStatus содержит информацию о текущем состоянии миграций БД.
type MigrationStatus struct {
	CurrentVersion uint `json:"currentVersion"`
	Dirty          bool `json:"dirty"`
	TotalAvailable int  `json:"totalAvailable"`
	UpToDate       bool `json:"upToDate"`
	SchemaTooNew   bool `json:"schemaTooNew"`
	Compatible     bool `json:"compatible"`
}

// MigrationCompatibilityError reports a schema state that the current binary
// must not operate against.
type MigrationCompatibilityError struct {
	CurrentVersion uint
	TotalAvailable int
	Dirty          bool
	SchemaTooNew   bool
}

func (e *MigrationCompatibilityError) Error() string {
	if e.SchemaTooNew {
		return fmt.Sprintf("database schema version %d is newer than embedded migrations %d", e.CurrentVersion, e.TotalAvailable)
	}
	if e.Dirty {
		return fmt.Sprintf("database schema version %d is dirty", e.CurrentVersion)
	}
	return "database schema is incompatible with this binary"
}

// Connect устанавливает подключение к базе данных PostgreSQL и возвращает обертку DB.
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	db, err := sql.Open("postgres", cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection pool: %w", err)
	}
	configureConnectionPool(db)

	ctx, cancel := context.WithTimeout(context.Background(), defaultStartupTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, operationTimeout: defaultOperationTimeout}, nil
}

// Query, QueryRow, Exec, Begin and Prepare keep the legacy repository API while
// ensuring that pool waits and SQL operations cannot block indefinitely. New
// code can pass a narrower deadline through the corresponding Context method.
func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	started := time.Now()
	ctx, cancel := db.withOperationTimeout(ctx)
	stopCancel := context.AfterFunc(ctx, cancel)
	rows, err := db.DB.QueryContext(ctx, query, args...)
	db.observe("database.query", started, err)
	if err != nil {
		stopCancel()
		cancel()
	}
	// database/sql keeps ctx alive until rows are closed. When callers forget to
	// close rows, the deadline still releases both the query and timer.
	return rows, err
}

func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	started := time.Now()
	ctx, cancel := db.withOperationTimeout(ctx)
	context.AfterFunc(ctx, cancel)
	row := db.DB.QueryRowContext(ctx, query, args...)
	// database/sql defers the actual row error until Scan. This duration still
	// captures dispatch and connection-pool wait; Scan failures are observed by
	// repository-level operation metrics.
	db.observe("database.query_row", started, nil)
	return row
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	started := time.Now()
	ctx, cancel := db.withOperationTimeout(ctx)
	defer cancel()
	result, err := db.DB.ExecContext(ctx, query, args...)
	db.observe("database.exec", started, err)
	return result, err
}

func (db *DB) Begin() (*sql.Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	started := time.Now()
	ctx, cancel := db.withOperationTimeout(ctx)
	stopCancel := context.AfterFunc(ctx, cancel)
	tx, err := db.DB.BeginTx(ctx, opts)
	db.observe("database.begin", started, err)
	if err != nil {
		stopCancel()
		cancel()
	}
	// The context must remain valid for the transaction lifetime. database/sql
	// rolls the transaction back automatically when the deadline is reached.
	return tx, err
}

func (db *DB) Prepare(query string) (*sql.Stmt, error) {
	return db.PrepareContext(context.Background(), query)
}

func (db *DB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	started := time.Now()
	ctx, cancel := db.withOperationTimeout(ctx)
	defer cancel()
	stmt, err := db.DB.PrepareContext(ctx, query)
	db.observe("database.prepare", started, err)
	return stmt, err
}

// SetMetrics attaches the application's optional in-process metrics registry.
// It is safe to call during startup before repositories begin serving requests.
func (db *DB) SetMetrics(metrics *observability.Registry) {
	if db == nil {
		return
	}
	db.metrics = metrics
}

func (db *DB) observe(name string, started time.Time, err error) {
	if db == nil || db.metrics == nil {
		return
	}
	db.metrics.Observe(name, time.Since(started), err)
	db.observePoolStats()
}

func (db *DB) observePoolStats() {
	stats := db.DB.Stats()
	db.poolMu.Lock()
	previous := db.lastPoolStats
	db.lastPoolStats = stats
	db.poolMu.Unlock()

	db.metrics.SetGauge("database.pool.max_open", float64(stats.MaxOpenConnections))
	db.metrics.SetGauge("database.pool.open", float64(stats.OpenConnections))
	db.metrics.SetGauge("database.pool.in_use", float64(stats.InUse))
	db.metrics.SetGauge("database.pool.idle", float64(stats.Idle))
	db.metrics.SetGauge("database.pool.wait_count_delta", float64(stats.WaitCount-previous.WaitCount))
	db.metrics.SetGauge("database.pool.wait_milliseconds_delta", float64((stats.WaitDuration-previous.WaitDuration).Microseconds())/1000)
}

func (db *DB) withOperationTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := db.operationTimeout
	if timeout <= 0 {
		timeout = defaultOperationTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func configureConnectionPool(db *sql.DB) {
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxIdleTime(defaultConnMaxIdleTime)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)
}

// RunMigrations применяет все доступные миграции к базе данных.
// Для DefaultMigrationsPath миграции берутся из embedded FS, чтобы собранное
// приложение не зависело от наличия исходной директории рядом с бинарником.
func (db *DB) RunMigrations(migrationsPath string) error {
	if err := db.CheckMigrationCompatibility(migrationsPath); err != nil {
		return err
	}

	m, err := db.newMigrator(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer closeMigrator(m)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// GetMigrationStatus возвращает текущую версию миграций и количество доступных миграций.
func (db *DB) GetMigrationStatus(migrationsPath string) (*MigrationStatus, error) {
	status := &MigrationStatus{}

	totalAvailable, err := countAvailableMigrations(migrationsPath)
	if err != nil {
		return nil, err
	}
	status.TotalAvailable = totalAvailable

	// Получение текущей версии из БД
	m, err := db.newMigrator(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer closeMigrator(m)

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	status.CurrentVersion = version
	status.Dirty = dirty
	status.applyCompatibility()

	return status, nil
}

// CheckMigrationCompatibility blocks startup/runtime operations for schema
// states that the current binary cannot safely understand.
func (db *DB) CheckMigrationCompatibility(migrationsPath string) error {
	status, err := db.GetMigrationStatus(migrationsPath)
	if err != nil {
		return err
	}
	if status.Compatible {
		return nil
	}
	if status.SchemaTooNew || status.Dirty {
		return &MigrationCompatibilityError{
			CurrentVersion: status.CurrentVersion,
			TotalAvailable: status.TotalAvailable,
			Dirty:          status.Dirty,
			SchemaTooNew:   status.SchemaTooNew,
		}
	}
	return nil
}

// RollbackMigration откатывает последнюю применённую миграцию (на 1 шаг назад).
func (db *DB) RollbackMigration(migrationsPath string) error {
	if err := db.CheckMigrationCompatibility(migrationsPath); err != nil {
		return err
	}

	m, err := db.newMigrator(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer closeMigrator(m)

	if err := m.Steps(-1); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	return nil
}

func (db *DB) newMigrator(migrationsPath string) (*migrate.Migrate, error) {
	if isDefaultMigrationsPath(migrationsPath) {
		sourceDriver, err := iofs.New(embeddedMigrations, embeddedMigrationsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedded migration source: %w", err)
		}

		driver, err := db.newMigrationDatabaseDriver()
		if err != nil {
			return nil, fmt.Errorf("failed to create migration driver: %w", err)
		}

		return migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	}

	if err := validateMigrationDirectory(migrationsPath); err != nil {
		return nil, err
	}

	driver, err := db.newMigrationDatabaseDriver()
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	return migrate.NewWithDatabaseInstance(
		"file://"+filepath.ToSlash(migrationsPath),
		"postgres",
		driver,
	)
}

func (db *DB) newMigrationDatabaseDriver() (*postgres.Postgres, error) {
	ctx := context.Background()
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithConnection(ctx, conn, &postgres.Config{})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return driver, nil
}

func countAvailableMigrations(migrationsPath string) (int, error) {
	entries, err := readMigrationDir(migrationsPath)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			total++
		}
	}
	return total, nil
}

func readMigrationDir(migrationsPath string) ([]fs.DirEntry, error) {
	if isDefaultMigrationsPath(migrationsPath) {
		entries, err := fs.ReadDir(embeddedMigrations, embeddedMigrationsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded migration directory: %w", err)
		}
		return entries, nil
	}

	if err := validateMigrationDirectory(migrationsPath); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}
	return entries, nil
}

func validateMigrationDirectory(migrationsPath string) error {
	info, err := os.Stat(migrationsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("migration directory %s not found", migrationsPath)
		}
		return fmt.Errorf("failed to check migration directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("migration path %s is not a directory", migrationsPath)
	}
	return nil
}

func isDefaultMigrationsPath(migrationsPath string) bool {
	return filepath.ToSlash(migrationsPath) == DefaultMigrationsPath
}

func closeMigrator(m *migrate.Migrate) {
	if m == nil {
		return
	}
	_, _ = m.Close()
}

func (s *MigrationStatus) applyCompatibility() {
	s.SchemaTooNew = int(s.CurrentVersion) > s.TotalAvailable
	s.UpToDate = int(s.CurrentVersion) == s.TotalAvailable && !s.Dirty
	s.Compatible = !s.Dirty && !s.SchemaTooNew
}

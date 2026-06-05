package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
)

const (
	defaultMaxOpenConns    = 5
	defaultMaxIdleConns    = 2
	defaultConnMaxIdleTime = 5 * time.Minute
	defaultConnMaxLifetime = 30 * time.Minute
)

// DB представляет собой обертку над подключением к базе данных SQL.
type DB struct {
	*sql.DB
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

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
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

		driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to create migration driver: %w", err)
		}

		return migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	}

	if err := validateMigrationDirectory(migrationsPath); err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	return migrate.NewWithDatabaseInstance(
		"file://"+filepath.ToSlash(migrationsPath),
		"postgres",
		driver,
	)
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

func (s *MigrationStatus) applyCompatibility() {
	s.SchemaTooNew = int(s.CurrentVersion) > s.TotalAvailable
	s.UpToDate = int(s.CurrentVersion) == s.TotalAvailable && !s.Dirty
	s.Compatible = !s.Dirty && !s.SchemaTooNew
}

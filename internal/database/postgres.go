package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	_ "github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
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
}

// Connect устанавливает подключение к базе данных PostgreSQL и возвращает обертку DB.
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	db, err := sql.Open("postgres", cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection pool: %w", err)
	}

	// We ping the database but do not return an error if it fails.
	// This allows the application to start and show a reconnect UI.
	if err := db.Ping(); err != nil {
		fmt.Printf("Warning: Failed to ping database on startup: %v\n", err)
	}

	return &DB{db}, nil
}

// getSourceURL normalizes the migration path to a source URL format.
func getSourceURL(migrationsPath string) string {
	if strings.HasPrefix(migrationsPath, "github://") || strings.HasPrefix(migrationsPath, "file://") {
		return migrationsPath
	}
	return "file://" + filepath.ToSlash(migrationsPath)
}

// RunMigrations применяет все доступные миграции из указанной директории к базе данных.
func (db *DB) RunMigrations(migrationsPath string) error {
	sourceURL := getSourceURL(migrationsPath)

	if !strings.HasPrefix(sourceURL, "github://") {
		// Проверка наличия директории миграций для файловой системы
		localPath := strings.TrimPrefix(sourceURL, "file://")
		info, err := os.Stat(localPath)
		if os.IsNotExist(err) {
			fmt.Printf("Migration directory %s not found. Skipping migrations.\n", localPath)
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to check migration directory: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("migration path %s is not a directory", localPath)
		}
	}

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres",
		driver,
	)
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
	sourceURL := getSourceURL(migrationsPath)

	if strings.HasPrefix(sourceURL, "github://") {
		src, err := source.Open(sourceURL)
		if err != nil {
			if os.IsNotExist(err) || strings.Contains(err.Error(), "404") {
				return status, nil
			}
			return nil, fmt.Errorf("failed to open github migration source: %w", err)
		}
		defer src.Close()

		version, err := src.First()
		if err == nil {
			status.TotalAvailable++
			for {
				version, err = src.Next(version)
				if err != nil {
					break
				}
				status.TotalAvailable++
			}
		} else if err != os.ErrNotExist {
			// Ignore if no migrations are found
		}
	} else {
		// Подсчёт доступных миграций (*.up.sql файлы)
		localPath := strings.TrimPrefix(sourceURL, "file://")
		entries, err := os.ReadDir(localPath)
		if err != nil {
			if os.IsNotExist(err) {
				return status, nil
			}
			return nil, fmt.Errorf("failed to read migration directory: %w", err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
				status.TotalAvailable++
			}
		}
	}

	// Получение текущей версии из БД
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	status.CurrentVersion = version
	status.Dirty = dirty
	status.UpToDate = int(version) >= status.TotalAvailable && !dirty

	return status, nil
}

// RollbackMigration откатывает последнюю применённую миграцию (на 1 шаг назад).
func (db *DB) RollbackMigration(migrationsPath string) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		getSourceURL(migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Steps(-1); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	return nil
}

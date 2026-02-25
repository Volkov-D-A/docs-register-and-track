package main

import (
	"fmt"
	"net/url"

	"docflow/internal/config"

	"github.com/golang-migrate/migrate/v4"
)

// ForceMigration applies force migration logic to the specified version.
func ForceMigration(cfg *config.DatabaseConfig, migrationsPath string, version int) error {
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		url.QueryEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.SSLMode,
	)

	m, err := migrate.New(
		"file://"+migrationsPath,
		dsn,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}

	return nil
}

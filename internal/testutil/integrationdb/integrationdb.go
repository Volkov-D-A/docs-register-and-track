// Package integrationdb provides an isolated, destructive PostgreSQL fixture
// for integration tests. It refuses to use a database whose name does not make
// its test-only purpose explicit.
package integrationdb

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	_ "github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
)

// Open resets the public schema and applies the application's embedded
// migrations. It skips tests outside the disposable integration environment.
func Open(t testing.TB) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DOCFLOW_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("set DOCFLOW_INTEGRATION_DSN to run PostgreSQL integration tests")
	}
	if err := ValidateDSN(dsn); err != nil {
		t.Fatalf("unsafe DOCFLOW_INTEGRATION_DSN: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("ping integration database: %v", err)
	}
	if _, err := db.Exec(`DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		_ = db.Close()
		t.Fatalf("reset integration schema: %v", err)
	}
	if err := (&database.DB{DB: db}).RunMigrations(database.DefaultMigrationsPath); err != nil {
		_ = db.Close()
		t.Fatalf("apply embedded migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ValidateDSN prevents a test run from clearing a development or production
// database. Supported URL and lib/pq key/value DSNs are accepted.
func ValidateDSN(dsn string) error {
	name, err := databaseName(dsn)
	if err != nil {
		return err
	}
	if strings.HasPrefix(name, "docflow_test") || strings.HasPrefix(name, "docflow_regression") {
		return nil
	}
	return fmt.Errorf("database name %q must start with docflow_test or docflow_regression", name)
}

func databaseName(dsn string) (string, error) {
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		name := strings.TrimPrefix(u.Path, "/")
		if name == "" {
			return "", fmt.Errorf("database name is empty")
		}
		return name, nil
	}
	for _, match := range regexp.MustCompile(`(?:^|\s)dbname=('[^']*'|"[^"]*"|[^\s]+)`).FindAllStringSubmatch(dsn, -1) {
		if len(match) == 2 {
			return strings.Trim(match[1], `"'`), nil
		}
	}
	return "", fmt.Errorf("database name not found")
}

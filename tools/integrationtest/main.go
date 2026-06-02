package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"
)

const (
	adminDSNEnv = "DOCFLOW_INTEGRATION_ADMIN_DSN"
	testDSNEnv  = "DOCFLOW_INTEGRATION_DSN"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "integration test runner failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	adminDSN := strings.TrimSpace(os.Getenv(adminDSNEnv))
	if adminDSN == "" {
		return fmt.Errorf("%s is required; point it to a maintenance database such as postgres, never to production data", adminDSNEnv)
	}

	prefix := strings.TrimSpace(os.Getenv("DOCFLOW_INTEGRATION_DB_PREFIX"))
	if prefix == "" {
		prefix = "docflow_test"
	}
	if !isSafeIntegrationDBName(prefix) {
		return fmt.Errorf("DOCFLOW_INTEGRATION_DB_PREFIX %q must start with docflow_test or docflow_regression", prefix)
	}

	dbName := fmt.Sprintf("%s_%d", sanitizeDBName(prefix), time.Now().UnixNano())
	testDSN, err := withDatabaseName(adminDSN, dbName)
	if err != nil {
		return err
	}

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return fmt.Errorf("open admin database: %w", err)
	}
	defer adminDB.Close()

	if err := createDatabase(adminDB, dbName); err != nil {
		return err
	}
	defer func() {
		if err := dropDatabase(adminDB, dbName); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to drop disposable integration database %s: %v\n", dbName, err)
		}
	}()

	cmd := exec.Command("go", "test", "./internal/repository", "-run", "Test(DocumentRegistration.*Integration|JournalRetentionFKIntegration|DatabaseConstraintsIntegration)", "-count=1", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), testDSNEnv+"="+testDSN)
	if cache := strings.TrimSpace(os.Getenv("GOCACHE")); cache != "" {
		cmd.Env = append(cmd.Env, "GOCACHE="+cache)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go integration tests: %w", err)
	}
	return nil
}

func createDatabase(db *sql.DB, dbName string) error {
	if !isSafeIntegrationDBName(dbName) {
		return fmt.Errorf("refusing to create unsafe database %q", dbName)
	}
	_, err := db.Exec(`CREATE DATABASE ` + pq.QuoteIdentifier(dbName))
	if err != nil {
		return fmt.Errorf("create disposable integration database %s: %w", dbName, err)
	}
	return nil
}

func dropDatabase(db *sql.DB, dbName string) error {
	if !isSafeIntegrationDBName(dbName) {
		return fmt.Errorf("refusing to drop unsafe database %q", dbName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, _ = db.ExecContext(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`, dbName)
	_, err := db.ExecContext(ctx, `DROP DATABASE IF EXISTS `+pq.QuoteIdentifier(dbName))
	return err
}

func withDatabaseName(dsn, dbName string) (string, error) {
	if !isSafeIntegrationDBName(dbName) {
		return "", fmt.Errorf("unsafe integration database name %q", dbName)
	}
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		u.Path = "/" + dbName
		return u.String(), nil
	}
	if regexp.MustCompile(`(?:^|\s)dbname=`).MatchString(dsn) {
		return regexp.MustCompile(`dbname=('[^']*'|"[^"]*"|[^\s]+)`).ReplaceAllString(dsn, "dbname="+dbName), nil
	}
	if strings.TrimSpace(dsn) == "" {
		return "", errors.New("empty admin DSN")
	}
	return strings.TrimSpace(dsn) + " dbname=" + dbName, nil
}

func isSafeIntegrationDBName(name string) bool {
	return strings.HasPrefix(name, "docflow_test") || strings.HasPrefix(name, "docflow_regression")
}

func sanitizeDBName(name string) string {
	clean := regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(name, "_")
	clean = strings.Trim(clean, "_")
	if clean == "" {
		return "docflow_test"
	}
	return clean
}

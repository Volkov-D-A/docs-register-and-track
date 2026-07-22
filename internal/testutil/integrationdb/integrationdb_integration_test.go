package integrationdb_test

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestEmbeddedMigrationsLifecycleIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}

	status, err := db.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil {
		t.Fatalf("read migration status: %v", err)
	}
	if !status.UpToDate || !status.Compatible || status.AvailableCount == 0 || status.LatestAvailableVersion == 0 {
		t.Fatalf("unexpected migrated status: %+v", status)
	}
	if err := db.RollbackMigration(database.DefaultMigrationsPath); err != nil {
		t.Fatalf("rollback latest embedded migration: %v", err)
	}
	if err := db.RunMigrations(database.DefaultMigrationsPath); err != nil {
		t.Fatalf("reapply embedded migrations: %v", err)
	}
	status, err = db.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil || !status.UpToDate || !status.Compatible {
		t.Fatalf("migration status after reapply: status=%+v err=%v", status, err)
	}
}

func TestValidateDSNIntegration(t *testing.T) {
	if err := integrationdb.ValidateDSN("postgres://user:pass@localhost/docflow_test_safe?sslmode=disable"); err != nil {
		t.Fatalf("safe dsn: %v", err)
	}
	if err := integrationdb.ValidateDSN("postgres://user:pass@localhost/docflow"); err == nil {
		t.Fatal("production-looking dsn was accepted")
	}
}

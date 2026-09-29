package integrationdb_test

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestEmbeddedMigrationsLifecycleIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)

	status, err := db.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil {
		t.Fatalf("read migration status: %v", err)
	}
	if !status.UpToDate || !status.Compatible || status.AvailableCount == 0 || status.LatestAvailableVersion == 0 {
		t.Fatalf("unexpected migrated status: %+v", status)
	}
	for step := 0; step < status.AvailableCount; step++ {
		if err := db.RollbackMigration(database.DefaultMigrationsPath); err != nil {
			t.Fatalf("rollback embedded migration step %d: %v", step, err)
		}
	}
	if err := db.RunMigrations(database.DefaultMigrationsPath); err != nil {
		t.Fatalf("reapply embedded migrations: %v", err)
	}
	status, err = db.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil || !status.UpToDate || !status.Compatible {
		t.Fatalf("migration status after reapply: status=%+v err=%v", status, err)
	}
}

func TestFreshSchemaExcludesUnusedHistoryIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	version, err := database.LatestSchemaVersion()
	if err != nil || version != 12 {
		t.Fatalf("expected migration 012, version=%d err=%v", version, err)
	}
	var remaining int
	if err := sqlDB.QueryRow(`SELECT count(*) FROM information_schema.columns WHERE table_schema = 'public' AND (table_name,column_name) IN (('system_settings','updated_at'),('storage_statistics','refresh_failed_at'),('assignment_recipients','viewed_at'),('organizations','created_at'),('resolution_executors','created_at'),('nomenclature','created_at'),('nomenclature','updated_at'),('departments','created_at'),('departments','updated_at'),('user_substitutions','created_by'),('user_substitutions','created_at'),('user_substitutions','updated_at'),('user_events','actor_user_id'),('user_events','entity_id'),('user_events','metadata'))`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("obsolete columns in fresh schema: count=%d err=%v", remaining, err)
	}
	if err := sqlDB.QueryRow(`SELECT count(*) FROM pg_class WHERE relnamespace = 'public'::regnamespace AND relname IN ('backup_jobs','backup_audit','idx_user_events_entity')`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("obsolete tables or index in fresh schema: count=%d err=%v", remaining, err)
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

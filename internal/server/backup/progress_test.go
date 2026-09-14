package backup

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
)

func TestProgressSurvivesReloadAndPreservesRetry(t *testing.T) {
	s := &Service{Directory: t.TempDir()}
	job := Job{ID: "a2b959cd-c532-4904-85d9-bf007a0d2467", State: "queued", CreatedAt: time.Now()}
	require.NoError(t, s.persist(&job))
	job.State = "transferring"
	require.NoError(t, s.persist(&job))
	require.NoError(t, s.persist(&job))
	job.State = "staged"
	job.Error = "transfer failed"
	require.NoError(t, s.persist(&job))
	job.State = "transferring"
	job.Error = ""
	require.NoError(t, s.persist(&job))
	jobs, err := s.Jobs()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, job.Stages, jobs[0].Stages)
	require.Len(t, jobs[0].Stages, 4)
	require.Equal(t, "transfer failed", jobs[0].Stages[2].Error)
	op := operation{BackupOperation: models.BackupOperation{ID: "a2b959cd-c532-4904-85d9-bf007a0d2469", CopyID: "a2b959cd-c532-4904-85d9-bf007a0d2468", Kind: "restore", State: "queued"}}
	require.NoError(t, s.persistOperation(&op))
	require.NoError(t, s.operationPhase(&op, "downloading", true))
	require.NoError(t, s.operationPhase(&op, "verifying", true))
	loaded, err := s.operation(op.ID)
	require.NoError(t, err)
	require.Equal(t, op.Stages, loaded.Stages)
	require.Len(t, loaded.Stages, 3)
}

func TestBackupAuditIntegration(t *testing.T) {
	dsn := os.Getenv("DOCFLOW_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("DOCFLOW_INTEGRATION_DSN is not set")
	}
	require.Contains(t, dsn, "docflow_test")
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TEMP TABLE users(id uuid PRIMARY KEY, full_name text)`)
	require.NoError(t, err)
	migration, err := os.ReadFile("../database/migrations/007_admin_audit_log.up.sql")
	require.NoError(t, err)
	// Apply the real table definition in the connection's temporary schema.
	_, err = db.Exec(strings.Replace(string(migration), "CREATE TABLE IF NOT EXISTS", "CREATE TEMP TABLE", 1))
	require.NoError(t, err)
	s := &Service{DB: db, Directory: t.TempDir()}
	job := Job{ID: "a2b959cd-c532-4904-85d9-bf007a0d2467", State: "completed", Actor: "schedule"}
	require.NoError(t, s.persist(&job))
	s.flushAudit(context.Background())
	s.flushAudit(context.Background())
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM admin_audit_log WHERE user_id IS NULL AND user_name='Расписание'`).Scan(&count))
	require.Equal(t, 1, count)
}

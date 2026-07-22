package app

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/outbox"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBackgroundLifecycleProcessesOutboxAfterMigrationIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	require.NoError(t, db.RollbackMigration(database.DefaultMigrationsPath))

	outboxRepo := repository.NewOutboxRepository(db)
	auditRepo := repository.NewAdminAuditLogRepository(db)
	worker := outbox.NewWorker(outboxRepo, nil, nil, auditRepo, nil, nil)
	lifecycle := newBackgroundLifecycle(db, worker, nil)
	lifecycle.SetApplicationContext(context.Background())
	lifecycle.ReconcileSchema()
	require.Error(t, lifecycle.CheckReady())

	require.NoError(t, db.RunMigrations(database.DefaultMigrationsPath))
	userID := uuid.New()
	_, err := db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active)
		VALUES ($1, $2, 'integration-hash', 'Lifecycle Integration', TRUE)`, userID, "lifecycle_"+uuid.NewString())
	require.NoError(t, err)

	payload, err := json.Marshal(models.CreateAdminAuditLogRequest{
		UserID:   userID,
		UserName: "Lifecycle Integration",
		Action:   "LIFECYCLE_INTEGRATION",
		Details:  "worker started after migration without application restart",
	})
	require.NoError(t, err)
	require.NoError(t, outboxRepo.Enqueue(models.OutboxEvent{
		EventType:        models.OutboxEventAudit,
		DeduplicationKey: "lifecycle-integration:" + uuid.NewString(),
		Payload:          string(payload),
	}))

	lifecycle.ReconcileSchema()
	require.Eventually(t, func() bool {
		var processed int
		err := db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE processed_at IS NOT NULL`).Scan(&processed)
		return err == nil && processed == 1
	}, 5*time.Second, 25*time.Millisecond)

	stopLifecycle(t, lifecycle)
}

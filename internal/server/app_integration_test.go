package server

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

type testStorage struct{}

func (testStorage) DeleteFile(context.Context, string) error                { return nil }
func (testStorage) RefreshStorageUsage(context.Context) (int, int64, error) { return 0, 0, nil }
func (testStorage) UploadFile(context.Context, string, io.Reader, int64, string) error {
	return nil
}
func (testStorage) DownloadFileToWriter(context.Context, string, io.Writer, int64) error {
	return nil
}

func TestServerProcessesOutboxWithoutWailsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	cfg := validConfig()
	cfg.Server.ListenAddress = "127.0.0.1:0"
	application, err := newWithDependencies(cfg, dependencies{
		connectDatabase: func(config.DatabaseConfig) (*database.DB, error) { return db, nil },
		newStorage:      func(config.MinioConfig) (serverStorage, error) { return testStorage{}, nil },
	})
	require.NoError(t, err)
	defer application.Close()

	userID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active)
		VALUES ($1, $2, 'integration-hash', 'Server Integration', TRUE)`, userID, "server_"+uuid.NewString())
	require.NoError(t, err)
	payload, err := json.Marshal(models.CreateAdminAuditLogRequest{
		UserID: userID, UserName: "Server Integration", Action: "SERVER_OUTBOX_INTEGRATION", Details: "processed without Wails",
	})
	require.NoError(t, err)
	require.NoError(t, repository.NewOutboxRepository(db).Enqueue(models.OutboxEvent{
		EventType: models.OutboxEventAudit, DeduplicationKey: "server-integration:" + uuid.NewString(), Payload: string(payload),
	}))

	runContext, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- application.Run(runContext) }()
	require.Eventually(t, func() bool {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = 'SERVER_OUTBOX_INTEGRATION'`).Scan(&count)
		return err == nil && count == 1
	}, 5*time.Second, 25*time.Millisecond)
	cancel()
	require.NoError(t, <-runDone)
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type compositionTestStorage struct{}

func (compositionTestStorage) UploadFile(context.Context, string, io.Reader, int64, string) error {
	return nil
}

func (compositionTestStorage) DownloadFileToWriter(context.Context, string, io.Writer, int64) error {
	return nil
}

func (compositionTestStorage) DeleteFile(context.Context, string) error { return nil }

func (compositionTestStorage) GetStorageInfo(context.Context) (int, string, error) {
	return 0, "0 B", nil
}

func (compositionTestStorage) RefreshStorageUsage(context.Context) (int, int64, error) {
	return 0, 0, nil
}

func TestCompositionRootStartupAndShutdownIntegration(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	var closeLoggerCalls atomic.Int32

	appOptions, failure := newWailsOptionsWithDependencies(
		&config.Config{},
		WailsOptionsParams{
			ConfigPath: "integration-config.json",
			ReleaseNotesSource: []byte(`version: 1.0.0
releasedAt: 2026-08-06
changes:
  - title: Integration test
    description: Composition root lifecycle
`),
			CloseLogger: func() { closeLoggerCalls.Add(1) },
		},
		wailsOptionsDependencies{
			connectDatabase: func(config.DatabaseConfig) (*database.DB, error) { return db, nil },
			newStorage: func(config.MinioConfig) (applicationStorage, error) {
				return compositionTestStorage{}, nil
			},
			newThemeService: services.NewThemeService,
		},
	)
	require.Nil(t, failure)
	require.NotNil(t, appOptions)
	require.NotNil(t, appOptions.OnStartup)
	require.NotNil(t, appOptions.OnShutdown)

	boundTypes := make([]string, 0, len(appOptions.Bind))
	for _, binding := range appOptions.Bind {
		boundTypes = append(boundTypes, fmt.Sprintf("%T", binding))
	}
	require.ElementsMatch(t, []string{
		"*services.AuthService",
		"*services.UserService",
		"*services.UserSubstitutionService",
		"*services.NomenclatureService",
		"*services.ReferenceService",
		"*services.DocumentAccessAdminService",
		"*services.DocumentKindService",
		"*services.DocumentQueryService",
		"*services.DocumentRegistrationService",
		"*services.AdministrativeOrderService",
		"*services.AssignmentService",
		"*services.DashboardService",
		"*services.StatisticsService",
		"*services.DepartmentService",
		"*services.SettingsService",
		"*services.AttachmentService",
		"*services.LinkService",
		"*services.AcknowledgmentService",
		"*services.SystemService",
		"*services.ReleaseNoteService",
		"*services.ThemeService",
		"*services.JournalService",
		"*services.AdminAuditLogService",
		"*services.UserEventService",
		"*services.OutboxAdminService",
	}, boundTypes)

	userID := uuid.New()
	_, err := db.Exec(`INSERT INTO users (id, login, password_hash, full_name, is_active)
		VALUES ($1, $2, 'integration-hash', 'Composition Root', TRUE)`, userID, "composition_"+uuid.NewString())
	require.NoError(t, err)
	payload, err := json.Marshal(models.CreateAdminAuditLogRequest{
		UserID:   userID,
		UserName: "Composition Root",
		Action:   "COMPOSITION_ROOT_INTEGRATION",
		Details:  "outbox worker started by Wails OnStartup",
	})
	require.NoError(t, err)
	require.NoError(t, repository.NewOutboxRepository(db).Enqueue(models.OutboxEvent{
		EventType:        models.OutboxEventAudit,
		DeduplicationKey: "composition-root:" + uuid.NewString(),
		Payload:          string(payload),
	}))

	startupContext, cancelStartup := context.WithCancel(context.Background())
	defer cancelStartup()
	appOptions.OnStartup(startupContext)
	require.Eventually(t, func() bool {
		var processed int
		err := db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = 'COMPOSITION_ROOT_INTEGRATION'`).Scan(&processed)
		return err == nil && processed == 1
	}, 5*time.Second, 25*time.Millisecond)

	appOptions.OnShutdown(context.Background())
	require.Equal(t, int32(1), closeLoggerCalls.Load())
	require.Error(t, sqlDB.Ping())
}

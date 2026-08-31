package app

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/background"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
)

const desktopMigrationStatusTimeout = 15 * time.Second

type migrationStatusClient interface {
	Status(context.Context) (*database.MigrationStatus, error)
}

// serverMigrationStatusReader keeps migration ownership on docflow-server.
// The path argument belongs to the shared server lifecycle contract and is
// deliberately ignored by the desktop adapter.
type serverMigrationStatusReader struct {
	client migrationStatusClient
}

func newServerMigrationStatusReader(client migrationStatusClient) *serverMigrationStatusReader {
	return &serverMigrationStatusReader{client: client}
}

func (r *serverMigrationStatusReader) GetMigrationStatus(string) (*database.MigrationStatus, error) {
	if r == nil || r.client == nil {
		return nil, errors.New("docflow-server migration status client is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), desktopMigrationStatusTimeout)
	defer cancel()
	return r.client.Status(ctx)
}

// Keep the existing composition-root names while the lifecycle implementation
// is shared by the desktop and the standalone server process.
type backgroundLifecycle = background.Lifecycle

func newBackgroundLifecycle(
	statusReader background.MigrationStatusReader,
	worker background.Worker,
	startupWork func(context.Context) error,
) *backgroundLifecycle {
	return background.NewLifecycle(statusReader, worker, startupWork)
}

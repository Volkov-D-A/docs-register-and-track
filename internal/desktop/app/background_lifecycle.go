package app

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

const desktopMigrationStatusTimeout = 15 * time.Second

type migrationStatusClient interface {
	Status(context.Context) (*dto.MigrationStatus, error)
}

// serverMigrationStatusReader keeps migration ownership on docflow-server.
type serverMigrationStatusReader struct {
	client migrationStatusClient
}

func newServerMigrationStatusReader(client migrationStatusClient) *serverMigrationStatusReader {
	return &serverMigrationStatusReader{client: client}
}

func (r *serverMigrationStatusReader) GetMigrationStatus() (*dto.MigrationStatus, error) {
	if r == nil || r.client == nil {
		return nil, errors.New("docflow-server migration status client is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), desktopMigrationStatusTimeout)
	defer cancel()
	return r.client.Status(ctx)
}

package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/background"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

// HTTP integration tests assemble repositories without starting the server's
// workers. They still need the real schema gate used by the production graph.
func newIntegrationManagementAPI(t *testing.T, app *App) *managementAPI {
	t.Helper()
	require.NotNil(t, app.db)
	require.Nil(t, app.lifecycle, "use newManagementAPI when supplying a lifecycle explicitly")
	app.lifecycle = background.NewLifecycle(func() (*dto.MigrationStatus, error) {
		return app.db.GetMigrationStatus(database.DefaultMigrationsPath)
	}, nil, nil)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		require.NoError(t, app.lifecycle.Stop(ctx))
	})
	app.lifecycle.ReconcileSchema()
	require.NoError(t, app.lifecycle.CheckReady())
	return newManagementAPI(app)
}

package app

import (
	"context"

	"github.com/Volkov-D-A/docs-register-and-track/internal/background"
)

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

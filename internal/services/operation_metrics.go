package services

import (
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
)

// measureOperation makes performance instrumentation explicit at the service
// boundary. Keeping names in source code prevents unbounded metric labels from
// request values such as IDs, filters, or SQL text.
func measureOperation[T any](metrics *observability.Registry, name string, operation func() (T, error)) (T, error) {
	started := time.Now()
	result, err := operation()
	if metrics != nil {
		metrics.Observe(name, time.Since(started), err)
	}
	return result, err
}

func measureOperationError(metrics *observability.Registry, name string, operation func() error) error {
	started := time.Now()
	err := operation()
	if metrics != nil {
		metrics.Observe(name, time.Since(started), err)
	}
	return err
}

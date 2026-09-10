package background

import (
	"context"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type stopWorker struct{}

func (stopWorker) Run(ctx context.Context) { <-ctx.Done() }
func TestLifecycleStopWaitsForStartupWriter(t *testing.T) {
	started := make(chan struct{})
	finish := make(chan struct{})
	lifecycle := NewLifecycle(func() (*dto.MigrationStatus, error) {
		return &dto.MigrationStatus{UpToDate: true, Compatible: true}, nil
	}, stopWorker{}, func(context.Context) error { close(started); <-finish; return nil })
	lifecycle.SetApplicationContext(context.Background())
	lifecycle.ReconcileSchema()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, lifecycle.Stop(ctx), context.DeadlineExceeded)
	close(finish)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	require.NoError(t, lifecycle.Stop(ctx2))
}

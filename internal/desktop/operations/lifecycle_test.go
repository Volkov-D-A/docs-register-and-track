package operations

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestLifecycleShutdownCancelsActiveOperation(t *testing.T) {
	lifecycle := NewLifecycle(time.Hour)
	opCtx, release := lifecycle.OperationContext()

	operationDone := make(chan struct{})
	go func() {
		<-opCtx.Done()
		release()
		close(operationDone)
	}()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := lifecycle.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown should wait for canceled operation: %v", err)
	}

	select {
	case <-operationDone:
	case <-time.After(time.Second):
		t.Fatal("active operation was not canceled")
	}
}

func TestLifecycleOperationAfterShutdownIsCanceled(t *testing.T) {
	lifecycle := NewLifecycle(time.Hour)
	if err := lifecycle.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown should complete: %v", err)
	}

	opCtx, release := lifecycle.OperationContext()
	defer release()

	if !errors.Is(opCtx.Err(), context.Canceled) {
		t.Fatalf("expected canceled operation context after shutdown, got %v", opCtx.Err())
	}
}

func TestLifecycleTimeoutAndRelease(t *testing.T) {
	lifecycle := NewLifecycle(10 * time.Millisecond)
	ctx, release := lifecycle.OperationContext()
	defer release()
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("timeout: %v", ctx.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("operation did not time out")
	}
	release()
	release() // Repeated cleanup must not decrement the wait group twice.
	if err := lifecycle.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleShutdownDeadlineThenDrain(t *testing.T) {
	lifecycle := NewLifecycle(0)
	operation, release := lifecycle.OperationContext()
	defer release()
	deadline, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancel()
	if err := lifecycle.Shutdown(deadline); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown must respect caller deadline: %v", err)
	}
	if !errors.Is(operation.Err(), context.Canceled) {
		t.Fatalf("operation remains active: %v", operation.Err())
	}
	release()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := lifecycle.Shutdown(ctx); err != nil {
		t.Fatalf("retry must drain: %v", err)
	}
}

func TestLifecycleConcurrentStartReleaseAndShutdown(t *testing.T) {
	lifecycle := NewLifecycle(time.Hour)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			<-start
			ctx, release := lifecycle.OperationContext()
			defer release()
			select {
			case <-ctx.Done():
				if !errors.Is(ctx.Err(), context.Canceled) {
					t.Errorf("operation cancellation: %v", ctx.Err())
				}
			case <-time.After(2 * time.Second):
				t.Error("operation did not stop")
			}
			release()
		})
	}
	for range 4 {
		workers.Go(func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := lifecycle.Shutdown(ctx); err != nil {
				t.Errorf("concurrent shutdown: %v", err)
			}
		})
	}
	close(start)
	workers.Wait()
}

func TestNilLifecycle(t *testing.T) {
	var lifecycle *Lifecycle
	ctx, release := lifecycle.OperationContext()
	release()
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	if _, ok := ctx.Deadline(); ok {
		t.Fatal("nil lifecycle must not introduce a deadline")
	}
	if err := lifecycle.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

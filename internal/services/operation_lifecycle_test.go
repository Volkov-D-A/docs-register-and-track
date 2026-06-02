package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOperationLifecycleShutdownCancelsActiveOperation(t *testing.T) {
	lifecycle := NewOperationLifecycle(time.Hour)
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

func TestOperationLifecycleOperationAfterShutdownIsCanceled(t *testing.T) {
	lifecycle := NewOperationLifecycle(time.Hour)
	if err := lifecycle.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown should complete: %v", err)
	}

	opCtx, release := lifecycle.OperationContext()
	defer release()

	if !errors.Is(opCtx.Err(), context.Canceled) {
		t.Fatalf("expected canceled operation context after shutdown, got %v", opCtx.Err())
	}
}

package services

import (
	"context"
	"testing"
	"time"
)

func TestServiceSetOperationLifecycle(t *testing.T) {
	lifecycle := NewOperationLifecycle(time.Second)
	defer func() {
		if err := lifecycle.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	tests := []struct {
		name string
		set  func(*OperationLifecycle)
		get  func() *OperationLifecycle
	}{
		{
			name: "attachment service",
			set:  (&AttachmentService{}).SetOperationLifecycle,
			get: func() *OperationLifecycle {
				service := &AttachmentService{}
				service.SetOperationLifecycle(lifecycle)
				return service.lifecycle
			},
		},
		{
			name: "document registration service",
			set:  (&DocumentRegistrationService{}).SetOperationLifecycle,
			get: func() *OperationLifecycle {
				service := &DocumentRegistrationService{}
				service.SetOperationLifecycle(lifecycle)
				return service.lifecycle
			},
		},
		{
			name: "journal service",
			set:  (&JournalService{}).SetOperationLifecycle,
			get: func() *OperationLifecycle {
				service := &JournalService{}
				service.SetOperationLifecycle(lifecycle)
				return service.lifecycle
			},
		},
		{
			name: "link service",
			set:  (&LinkService{}).SetOperationLifecycle,
			get: func() *OperationLifecycle {
				service := &LinkService{}
				service.SetOperationLifecycle(lifecycle)
				return service.lifecycle
			},
		},
		{
			name: "statistics service",
			set:  (&StatisticsService{}).SetOperationLifecycle,
			get: func() *OperationLifecycle {
				service := &StatisticsService{}
				service.SetOperationLifecycle(lifecycle)
				return service.lifecycle
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.set(nil)
			if got := tt.get(); got != lifecycle {
				t.Fatalf("expected lifecycle to be assigned")
			}
		})
	}
}

func TestSystemServiceStartupStoresContext(t *testing.T) {
	service := NewSystemService(nil)
	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	service.Startup(ctx)

	if service.db != nil {
		t.Fatal("expected nil db to be preserved")
	}
	if service.ctx != ctx {
		t.Fatal("expected startup context to be stored")
	}
}

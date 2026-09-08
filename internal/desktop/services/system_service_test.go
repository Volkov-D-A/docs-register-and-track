package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

type fakeSystemClient struct {
	compatibility *dto.CompatibilityResult
	status        *dto.SystemStatus
	err           error
}

func (c fakeSystemClient) Compatibility(context.Context, string) (*dto.CompatibilityResult, error) {
	return c.compatibility, c.err
}
func (c fakeSystemClient) SystemStatus(context.Context) (*dto.SystemStatus, error) {
	return c.status, c.err
}

func TestSystemServiceAllowsReadyAndMaintenanceServers(t *testing.T) {
	for _, state := range []string{"ready", "maintenance"} {
		service, _ := NewSystemService(fakeSystemClient{
			compatibility: &dto.CompatibilityResult{Compatible: true, Code: "compatible"},
			status:        &dto.SystemStatus{Status: state, Code: state},
		}, "1.0.6")

		result := service.GetBootstrapStatus()
		assert.Equal(t, state, result.State)
		assert.NotNil(t, result.System)
	}
}

func TestSystemServiceBlocksIncompatibleClient(t *testing.T) {
	service, _ := NewSystemService(fakeSystemClient{
		compatibility: &dto.CompatibilityResult{Compatible: false, Code: "client_too_old"},
	}, "1.0.5")

	result := service.GetBootstrapStatus()
	assert.Equal(t, "client_too_old", result.State)
	assert.Nil(t, result.System)
}

func TestSystemServiceReturnsActionableConnectionFailure(t *testing.T) {
	service, _ := NewSystemService(fakeSystemClient{err: &serverclient.SystemRequestError{
		Kind: serverclient.SystemErrorUnavailable,
		Err:  errors.New("connection refused"),
	}}, "1.0.6")

	result := service.GetBootstrapStatus()
	assert.Equal(t, "server_unavailable", result.State)
	assert.Contains(t, result.Message, "Сервер недоступен")
	assert.NotContains(t, result.Message, "connection refused")
}

type contextSystemClient struct {
	fakeSystemClient
	check func(context.Context, string) (*dto.CompatibilityResult, error)
}

func (c contextSystemClient) Compatibility(ctx context.Context, version string) (*dto.CompatibilityResult, error) {
	return c.check(ctx, version)
}

func TestSystemServiceStartupContext(t *testing.T) {
	type contextKey struct{}
	parent, cancelParent := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "wails"))
	defer cancelParent()
	var request context.Context
	var started bool
	client := contextSystemClient{
		fakeSystemClient: fakeSystemClient{status: &dto.SystemStatus{Status: "ready"}},
		check: func(ctx context.Context, version string) (*dto.CompatibilityResult, error) {
			request = ctx
			assert.Equal(t, "1.0.6", version)
			deadline, ok := ctx.Deadline()
			assert.True(t, ok)
			assert.InDelta(t, 15, time.Until(deadline).Seconds(), 1)
			if started {
				assert.Equal(t, "wails", ctx.Value(contextKey{}))
				assert.ErrorIs(t, ctx.Err(), context.Canceled)
				return nil, &serverclient.SystemRequestError{Kind: serverclient.SystemErrorUnavailable, Err: ctx.Err()}
			}
			assert.NoError(t, ctx.Err())
			return &dto.CompatibilityResult{Compatible: true}, nil
		},
	}
	service, start := NewSystemService(client, "1.0.6")
	assert.Equal(t, "ready", service.GetBootstrapStatus().State)
	assert.ErrorIs(t, request.Err(), context.Canceled)
	start(parent)
	started = true
	cancelParent()
	assert.Equal(t, serverclient.SystemErrorUnavailable, service.GetBootstrapStatus().State)
}

func TestSystemServiceConcurrentStartupAndRequests(t *testing.T) {
	service, start := NewSystemService(fakeSystemClient{
		compatibility: &dto.CompatibilityResult{Compatible: true},
		status:        &dto.SystemStatus{Status: "ready"},
	}, "1.0.6")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			start(context.Background())
		}
	}()
	for i := 0; i < 100; i++ {
		assert.Equal(t, "ready", service.GetBootstrapStatus().State)
	}
	wg.Wait()
}

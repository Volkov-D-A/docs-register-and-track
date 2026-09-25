package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/stretchr/testify/require"
)

type dashboardClientStub struct {
	read func(context.Context) (*dto.DashboardActivity, error)
}

func (c dashboardClientStub) GetDashboardActivity(ctx context.Context) (*dto.DashboardActivity, error) {
	return c.read(ctx)
}

type outboxAdminClientStub struct {
	serverclient.OutboxAdminClient
	requeue func(context.Context, string) error
}

func (c outboxAdminClientStub) RequeueOutboxEvent(ctx context.Context, id string) error {
	return c.requeue(ctx, id)
}

func TestDashboardAdapterReturnsServerScopeAndCancelsRequest(t *testing.T) {
	var requestContext context.Context
	want := &dto.DashboardActivity{ExpiringAssignments: []dto.DashboardAssignment{{ID: "substituted-assignment"}}}
	service := NewDashboardService(dashboardClientStub{read: func(ctx context.Context) (*dto.DashboardActivity, error) {
		requestContext = ctx
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		require.NoError(t, ctx.Err())
		return want, nil
	}})
	result, err := service.GetActivity()
	require.NoError(t, err)
	require.Same(t, want, result)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}

func TestOutboxAdapterLeavesValidationAndPermissionsToServer(t *testing.T) {
	var requestContext context.Context
	service := NewOutboxAdminService(outboxAdminClientStub{requeue: func(ctx context.Context, id string) error {
		requestContext = ctx
		require.Equal(t, "invalid-id", id)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		require.NoError(t, ctx.Err())
		return models.ErrForbidden
	}})
	require.ErrorIs(t, service.Requeue("invalid-id"), models.ErrForbidden)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}

func TestDashboardOutboxAdaptersRejectMissingClients(t *testing.T) {
	_, err := NewDashboardService(nil).GetActivity()
	require.ErrorIs(t, err, errDashboardServiceClientNotConfigured)
	service := NewOutboxAdminService(nil)
	stats, err := service.GetStats()
	require.Equal(t, models.OutboxStats{}, stats)
	require.ErrorIs(t, err, errOutboxAdminServiceClientNotConfigured)
	_, err = service.GetFailed(50)
	require.ErrorIs(t, err, errOutboxAdminServiceClientNotConfigured)
	require.ErrorIs(t, service.Requeue(""), errOutboxAdminServiceClientNotConfigured)
}

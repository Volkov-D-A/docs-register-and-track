package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
)

type workspaceClientStub struct {
	read func(context.Context, string, string) (*dto.WorkspaceOverview, error)
}

func (c workspaceClientStub) GetWorkspaceOverview(ctx context.Context, assignmentMode, acknowledgmentMode string) (*dto.WorkspaceOverview, error) {
	return c.read(ctx, assignmentMode, acknowledgmentMode)
}

func (c workspaceClientStub) ListWorkspaceAcknowledgments(context.Context, string, int, int) (*dto.PagedResult[dto.WorkspaceAcknowledgment], error) {
	return &dto.PagedResult[dto.WorkspaceAcknowledgment]{}, nil
}

type outboxAdminClientStub struct {
	serverclient.OutboxAdminClient
	requeue func(context.Context, string) error
}

func (c outboxAdminClientStub) RequeueOutboxEvent(ctx context.Context, id string) error {
	return c.requeue(ctx, id)
}

func TestWorkspaceAdapterReturnsServerScopeAndCancelsRequest(t *testing.T) {
	var requestContext context.Context
	want := &dto.WorkspaceOverview{Assignments: []dto.WorkspaceAssignment{{ID: "substituted-assignment"}}}
	service := NewWorkspaceService(workspaceClientStub{read: func(ctx context.Context, assignmentMode, acknowledgmentMode string) (*dto.WorkspaceOverview, error) {
		requestContext = ctx
		require.Equal(t, "execution", assignmentMode)
		require.Equal(t, "control", acknowledgmentMode)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		require.NoError(t, ctx.Err())
		return want, nil
	}})
	result, err := service.GetOverview("execution", "control")
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

func TestWorkspaceOutboxAdaptersRejectMissingClients(t *testing.T) {
	_, err := NewWorkspaceService(nil).GetOverview("", "")
	require.ErrorIs(t, err, errWorkspaceServiceClientNotConfigured)
	service := NewOutboxAdminService(nil)
	stats, err := service.GetStats()
	require.Equal(t, models.OutboxStats{}, stats)
	require.ErrorIs(t, err, errOutboxAdminServiceClientNotConfigured)
	_, err = service.GetFailed(50)
	require.ErrorIs(t, err, errOutboxAdminServiceClientNotConfigured)
	require.ErrorIs(t, service.Requeue(""), errOutboxAdminServiceClientNotConfigured)
}

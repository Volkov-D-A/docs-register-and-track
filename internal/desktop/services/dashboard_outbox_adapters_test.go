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
	read      func(context.Context, string) (*dto.WorkspaceOverview, error)
	documents func(context.Context) (*dto.WorkspaceDocuments, error)
}

func (c workspaceClientStub) GetWorkspaceOverview(ctx context.Context, assignmentMode string) (*dto.WorkspaceOverview, error) {
	return c.read(ctx, assignmentMode)
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
	service := NewWorkspaceService(workspaceClientStub{read: func(ctx context.Context, assignmentMode string) (*dto.WorkspaceOverview, error) {
		requestContext = ctx
		require.Equal(t, "execution", assignmentMode)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		require.NoError(t, ctx.Err())
		return want, nil
	}})
	result, err := service.GetOverview("execution")
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
	_, err := NewWorkspaceService(nil).GetOverview("")
	require.ErrorIs(t, err, errWorkspaceServiceClientNotConfigured)
	service := NewOutboxAdminService(nil)
	stats, err := service.GetStats()
	require.Equal(t, models.OutboxStats{}, stats)
	require.ErrorIs(t, err, errOutboxAdminServiceClientNotConfigured)
	_, err = service.GetFailed(50)
	require.ErrorIs(t, err, errOutboxAdminServiceClientNotConfigured)
	require.ErrorIs(t, service.Requeue(""), errOutboxAdminServiceClientNotConfigured)
}

func (c workspaceClientStub) GetWorkspaceDocuments(ctx context.Context) (*dto.WorkspaceDocuments, error) {
	if c.documents != nil {
		return c.documents(ctx)
	}
	return &dto.WorkspaceDocuments{}, nil
}

func TestWorkspaceRecentDocumentsAdapterReturnsPreviewAndCancelsContext(t *testing.T) {
	var requestContext context.Context
	want := &dto.WorkspaceDocuments{Available: true, Items: []dto.WorkspaceDocument{{ID: "doc"}}}
	service := NewWorkspaceService(workspaceClientStub{documents: func(ctx context.Context) (*dto.WorkspaceDocuments, error) {
		requestContext = ctx
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		return want, nil
	}})
	result, err := service.GetRecentDocuments()
	require.NoError(t, err)
	require.Same(t, want, result)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
	_, err = NewWorkspaceService(nil).GetRecentDocuments()
	require.ErrorIs(t, err, errWorkspaceServiceClientNotConfigured)
}

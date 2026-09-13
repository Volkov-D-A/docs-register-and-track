package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
)

type journalClientStub struct {
	read func(context.Context, string) ([]dto.JournalEntry, error)
}

func (c journalClientStub) GetDocumentJournal(ctx context.Context, id string) ([]dto.JournalEntry, error) {
	return c.read(ctx, id)
}

type adminAuditClientStub struct {
	read func(context.Context, int, int) (*dto.AdminAuditLogPage, error)
}

func (c adminAuditClientStub) GetAdminAuditLog(ctx context.Context, page, size int) (*dto.AdminAuditLogPage, error) {
	return c.read(ctx, page, size)
}

func TestJournalAdapterPreservesPermissionErrorAndCancelsRequest(t *testing.T) {
	var requestContext context.Context
	denied := models.ErrForbidden
	service := NewJournalService(journalClientStub{read: func(ctx context.Context, id string) ([]dto.JournalEntry, error) {
		requestContext = ctx
		require.Equal(t, "document", id)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		require.NoError(t, ctx.Err())
		return nil, denied
	}})
	rows, err := service.GetByDocumentID("document")
	require.Nil(t, rows)
	require.ErrorIs(t, err, denied)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}

func TestAdminAuditAdapterLeavesPaginationToServer(t *testing.T) {
	var requestContext context.Context
	want := &dto.AdminAuditLogPage{Page: 1, Total: 12}
	service := NewAdminAuditLogService(adminAuditClientStub{read: func(ctx context.Context, page, size int) (*dto.AdminAuditLogPage, error) {
		requestContext = ctx
		require.Equal(t, 0, page)
		require.Equal(t, 200, size)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		return want, nil
	}})
	result, err := service.GetAll(0, 200)
	require.NoError(t, err)
	require.Same(t, want, result)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}

func TestJournalAdaptersRejectMissingClients(t *testing.T) {
	_, err := NewJournalService(nil).GetByDocumentID("document")
	require.ErrorIs(t, err, errJournalServiceClientNotConfigured)
	_, err = NewAdminAuditLogService(nil).GetAll(1, 50)
	require.ErrorIs(t, err, errAdminAuditLogServiceClientNotConfigured)
}

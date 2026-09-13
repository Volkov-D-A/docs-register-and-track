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

type acknowledgmentClientStub struct {
	serverclient.AcknowledgmentClient
	create  func(context.Context, string, string, []string) (*dto.Acknowledgment, error)
	pending func(context.Context, string) ([]dto.Acknowledgment, error)
	confirm func(context.Context, string) error
}

func (c acknowledgmentClientStub) CreateAcknowledgment(ctx context.Context, document, content string, users []string) (*dto.Acknowledgment, error) {
	return c.create(ctx, document, content, users)
}
func (c acknowledgmentClientStub) ListPendingAcknowledgmentsByDocument(ctx context.Context, document string) ([]dto.Acknowledgment, error) {
	return c.pending(ctx, document)
}
func (c acknowledgmentClientStub) MarkAcknowledgmentConfirmed(ctx context.Context, id string) error {
	return c.confirm(ctx, id)
}

func TestAcknowledgmentAdapterPreservesRecipientsAndBoundsRequest(t *testing.T) {
	var requestContext context.Context
	want := &dto.Acknowledgment{ID: "server-id"}
	client := acknowledgmentClientStub{create: func(ctx context.Context, document, content string, users []string) (*dto.Acknowledgment, error) {
		requestContext = ctx
		require.Equal(t, "document", document)
		require.Equal(t, "  Ознакомиться  ", content)
		require.Equal(t, []string{"recipient", "substitute"}, users)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		require.NoError(t, ctx.Err())
		return want, nil
	}}
	got, err := NewAcknowledgmentService(client).Create("document", "  Ознакомиться  ", []string{"recipient", "substitute"})
	require.NoError(t, err)
	require.Same(t, want, got)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}

func TestAcknowledgmentAdapterUsesServerPendingRowsAndPermissionErrors(t *testing.T) {
	want := []dto.Acknowledgment{{ID: "principal-row", DocumentID: "document"}}
	denied := models.NewForbidden("ознакомление не назначено пользователю")
	client := acknowledgmentClientStub{
		pending: func(_ context.Context, document string) ([]dto.Acknowledgment, error) {
			require.Equal(t, "document", document)
			return want, nil
		},
		confirm: func(_ context.Context, id string) error {
			require.Equal(t, "principal-row", id)
			return denied
		},
	}
	service := NewAcknowledgmentService(client)
	got, err := service.GetCurrentUserPendingByDocument("document")
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.ErrorIs(t, service.MarkConfirmed("principal-row"), denied)
}

func TestAcknowledgmentAdapterMissingClient(t *testing.T) {
	service := NewAcknowledgmentService(nil)
	_, err := service.Create("", "", nil)
	require.ErrorIs(t, err, errAcknowledgmentClientNotConfigured)
	_, err = service.GetList("")
	require.ErrorIs(t, err, errAcknowledgmentClientNotConfigured)
	_, err = service.GetPendingForCurrentUser()
	require.ErrorIs(t, err, errAcknowledgmentClientNotConfigured)
	_, err = service.GetCurrentUserPendingByDocument("")
	require.ErrorIs(t, err, errAcknowledgmentClientNotConfigured)
	_, err = service.GetAllActive()
	require.ErrorIs(t, err, errAcknowledgmentClientNotConfigured)
	require.ErrorIs(t, service.MarkViewed(""), errAcknowledgmentClientNotConfigured)
	require.ErrorIs(t, service.MarkConfirmed(""), errAcknowledgmentClientNotConfigured)
	require.ErrorIs(t, service.Delete(""), errAcknowledgmentClientNotConfigured)
}

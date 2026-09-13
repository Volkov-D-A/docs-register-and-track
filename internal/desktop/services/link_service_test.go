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

type linkClientStub struct {
	serverclient.LinkClient
	create func(context.Context, string, string, string) (*dto.DocumentLink, error)
}

func (c linkClientStub) LinkDocuments(ctx context.Context, source, target, kind string) (*dto.DocumentLink, error) {
	return c.create(ctx, source, target, kind)
}

func TestLinkAdapterPreservesCancellationCommandAndServerError(t *testing.T) {
	var requestContext context.Context
	denied := models.NewForbidden("недоступен целевой приказ")
	client := linkClientStub{create: func(ctx context.Context, source, target, kind string) (*dto.DocumentLink, error) {
		requestContext = ctx
		require.Equal(t, "source", source)
		require.Equal(t, "target", target)
		require.Equal(t, "order_cancels", kind)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
		require.NoError(t, ctx.Err())
		return nil, denied
	}}
	result, err := NewLinkService(client).LinkDocuments("source", "target", "order_cancels")
	require.Nil(t, result)
	require.ErrorIs(t, err, denied)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}

func TestLinkAdapterMissingClient(t *testing.T) {
	service := NewLinkService(nil)
	_, err := service.LinkDocuments("", "", "")
	require.ErrorIs(t, err, errLinkServiceClientNotConfigured)
	_, err = service.GetDocumentLinks("")
	require.ErrorIs(t, err, errLinkServiceClientNotConfigured)
	_, err = service.GetDocumentFlow("")
	require.ErrorIs(t, err, errLinkServiceClientNotConfigured)
	require.ErrorIs(t, service.UnlinkDocument(""), errLinkServiceClientNotConfigured)
}

package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
)

type administrativeOrderClientStub struct {
	mark func(context.Context, string) (*dto.AdministrativeOrderAcknowledgmentPerson, error)
}

func (c administrativeOrderClientStub) MarkAdministrativeOrderAcknowledged(ctx context.Context, id string) (*dto.AdministrativeOrderAcknowledgmentPerson, error) {
	return c.mark(ctx, id)
}

func TestAdministrativeOrderAdapterReturnsServerAcknowledgment(t *testing.T) {
	want := &dto.AdministrativeOrderAcknowledgmentPerson{ID: "person"}
	denied := models.NewForbidden("нет права изменения приказа")
	for _, serverErr := range []error{nil, denied} {
		var requestContext context.Context
		client := administrativeOrderClientStub{mark: func(ctx context.Context, id string) (*dto.AdministrativeOrderAcknowledgmentPerson, error) {
			requestContext = ctx
			require.Equal(t, "person", id)
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			require.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
			require.NoError(t, ctx.Err())
			if serverErr != nil {
				return nil, serverErr
			}
			return want, nil
		}}
		result, err := NewAdministrativeOrderService(client).MarkAcknowledged("person")
		require.ErrorIs(t, err, serverErr)
		if serverErr == nil {
			require.Same(t, want, result)
		} else {
			require.Nil(t, result)
		}
		require.ErrorIs(t, requestContext.Err(), context.Canceled)
	}
}

func TestAdministrativeOrderAdapterMissingClient(t *testing.T) {
	_, err := NewAdministrativeOrderService(nil).MarkAcknowledged("")
	require.ErrorIs(t, err, errAdministrativeOrderServiceClientNotConfigured)
}

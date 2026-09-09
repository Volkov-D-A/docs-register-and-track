package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
)

type userEventClientStub struct {
	ctx    context.Context
	filter models.UserEventFilter
	id     string
	err    error
}

func (c *userEventClientStub) ListUserEvents(ctx context.Context, f models.UserEventFilter) (*dto.PagedResult[dto.UserEvent], error) {
	c.ctx, c.filter = ctx, f
	return &dto.PagedResult[dto.UserEvent]{TotalCount: 3}, c.err
}
func (c *userEventClientStub) GetUnreadUserEventCount(ctx context.Context) (int, error) {
	c.ctx = ctx
	return 3, c.err
}
func (c *userEventClientStub) MarkUserEventRead(ctx context.Context, id string) error {
	c.ctx, c.id = ctx, id
	return c.err
}
func (c *userEventClientStub) MarkDocumentUserEventsRead(ctx context.Context, id string) error {
	c.ctx, c.id = ctx, id
	return c.err
}
func (c *userEventClientStub) MarkAllUserEventsRead(ctx context.Context) error {
	c.ctx = ctx
	return c.err
}

func TestUserEventAdapter(t *testing.T) {
	c := &userEventClientStub{}
	s := NewUserEventService(c)
	filter := models.UserEventFilter{Page: 2, PageSize: 7, UnreadOnly: true}
	result, err := s.GetCurrentUserEvents(filter)
	require.NoError(t, err)
	require.Equal(t, 3, result.TotalCount)
	require.Equal(t, filter, c.filter)
	count, err := s.GetUnreadCount()
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.NoError(t, s.MarkRead("event"))
	require.Equal(t, "event", c.id)
	require.NoError(t, s.MarkDocumentRead("document"))
	require.Equal(t, "document", c.id)
	require.NoError(t, s.MarkAllRead())
	_, deadline := c.ctx.Deadline()
	require.True(t, deadline)
	require.ErrorIs(t, c.ctx.Err(), context.Canceled)
	c.err = errors.New("HTTP failure")
	_, err = s.GetCurrentUserEvents(filter)
	require.ErrorIs(t, err, c.err)
	_, err = s.GetUnreadCount()
	require.ErrorIs(t, err, c.err)
	require.ErrorIs(t, s.MarkRead("event"), c.err)
	require.ErrorIs(t, s.MarkDocumentRead("document"), c.err)
	require.ErrorIs(t, s.MarkAllRead(), c.err)
}

func TestUserEventAdapterMissingClient(t *testing.T) {
	s := NewUserEventService(nil)
	_, err := s.GetCurrentUserEvents(models.UserEventFilter{})
	require.ErrorIs(t, err, errUserEventClientNotConfigured)
	_, err = s.GetUnreadCount()
	require.ErrorIs(t, err, errUserEventClientNotConfigured)
	require.ErrorIs(t, s.MarkRead("event"), errUserEventClientNotConfigured)
	require.ErrorIs(t, s.MarkDocumentRead("document"), errUserEventClientNotConfigured)
	require.ErrorIs(t, s.MarkAllRead(), errUserEventClientNotConfigured)
}

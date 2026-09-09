package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// UserEventService exposes the current user's events through the HTTP API.
type UserEventService struct{ server serverclient.UserEventClient }

func NewUserEventService(client serverclient.UserEventClient) *UserEventService {
	return &UserEventService{server: client}
}

var errUserEventClientNotConfigured = errors.New("docflow-server user event client is not configured")

func userEventClientContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func (s *UserEventService) GetCurrentUserEvents(filter models.UserEventFilter) (*dto.PagedResult[dto.UserEvent], error) {
	if s.server == nil {
		return nil, errUserEventClientNotConfigured
	}

	ctx, cancel := userEventClientContext()
	defer cancel()
	return s.server.ListUserEvents(ctx, filter)
}

func (s *UserEventService) GetUnreadCount() (int, error) {
	if s.server == nil {
		return 0, errUserEventClientNotConfigured
	}

	ctx, cancel := userEventClientContext()
	defer cancel()
	return s.server.GetUnreadUserEventCount(ctx)
}

func (s *UserEventService) MarkRead(id string) error {
	if s.server == nil {
		return errUserEventClientNotConfigured
	}

	ctx, cancel := userEventClientContext()
	defer cancel()
	return s.server.MarkUserEventRead(ctx, id)
}

func (s *UserEventService) MarkDocumentRead(documentID string) error {
	if s.server == nil {
		return errUserEventClientNotConfigured
	}

	ctx, cancel := userEventClientContext()
	defer cancel()
	return s.server.MarkDocumentUserEventsRead(ctx, documentID)
}

func (s *UserEventService) MarkAllRead() error {
	if s.server == nil {
		return errUserEventClientNotConfigured
	}

	ctx, cancel := userEventClientContext()
	defer cancel()
	return s.server.MarkAllUserEventsRead(ctx)
}

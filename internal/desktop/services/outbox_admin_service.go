package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
)

// OutboxAdminService exposes server-owned operations through HTTP.
type OutboxAdminService struct {
	server serverclient.OutboxAdminClient
}

func NewOutboxAdminService(client serverclient.OutboxAdminClient) *OutboxAdminService {
	return &OutboxAdminService{server: client}
}

var errOutboxAdminServiceClientNotConfigured = errors.New("docflow-server outbox admin client is not configured")

func (s *OutboxAdminService) GetStats() (models.OutboxStats, error) {
	if s.server == nil {
		return models.OutboxStats{}, errOutboxAdminServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetOutboxStats(ctx)
}

func (s *OutboxAdminService) GetFailed(limit int) ([]models.FailedOutboxEvent, error) {
	if s.server == nil {
		return nil, errOutboxAdminServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetFailedOutboxEvents(ctx, limit)
}

func (s *OutboxAdminService) Requeue(id string) error {
	if s.server == nil {
		return errOutboxAdminServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.RequeueOutboxEvent(ctx, id)
}

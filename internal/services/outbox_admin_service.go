package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// OutboxAdminService exposes operational state without granting the UI direct
// database access. Every operation requires the existing administrator right.
type OutboxAdminService struct {
	repo   OutboxAdminStore
	auth   SystemPermissionPrincipal
	server serverclient.OutboxAdminClient
}

type OutboxAdminStore interface {
	Stats() (models.OutboxStats, error)
	GetFailed(int) ([]models.FailedOutboxEvent, error)
	Requeue(uuid.UUID) error
}

func NewOutboxAdminService(repo OutboxAdminStore, auth SystemPermissionPrincipal) *OutboxAdminService {
	return &OutboxAdminService{repo: repo, auth: auth}
}

func NewOutboxAdminServiceWithClient(client serverclient.OutboxAdminClient) *OutboxAdminService {
	return &OutboxAdminService{server: client}
}

func outboxAdminClientContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func (s *OutboxAdminService) GetStats() (models.OutboxStats, error) {
	if s.server != nil {
		ctx, cancel := outboxAdminClientContext()
		defer cancel()
		return s.server.GetOutboxStats(ctx)
	}
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return models.OutboxStats{}, err
	}
	return s.repo.Stats()
}

func (s *OutboxAdminService) GetFailed(limit int) ([]models.FailedOutboxEvent, error) {
	if s.server != nil {
		ctx, cancel := outboxAdminClientContext()
		defer cancel()
		return s.server.GetFailedOutboxEvents(ctx, limit)
	}
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	return s.repo.GetFailed(limit)
}

func (s *OutboxAdminService) Requeue(id string) error {
	if s.server != nil {
		ctx, cancel := outboxAdminClientContext()
		defer cancel()
		return s.server.RequeueOutboxEvent(ctx, id)
	}
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return err
	}
	eventID, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID outbox-задачи", err)
	}
	return s.repo.Requeue(eventID)
}

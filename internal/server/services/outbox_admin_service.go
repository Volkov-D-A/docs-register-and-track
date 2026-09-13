package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

// OutboxAdminService provides server-side outbox administration.
// Every operation requires the existing administrator right.
type OutboxAdminService struct {
	repo ports.OutboxAdminStore
	auth ports.SystemPermissionPrincipal
}

func NewOutboxAdminService(repo ports.OutboxAdminStore, auth ports.SystemPermissionPrincipal) *OutboxAdminService {
	return &OutboxAdminService{repo: repo, auth: auth}
}

func (s *OutboxAdminService) GetStats() (models.OutboxStats, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return models.OutboxStats{}, err
	}
	return s.repo.Stats()
}

func (s *OutboxAdminService) GetFailed(limit int) ([]models.FailedOutboxEvent, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	return s.repo.GetFailed(limit)
}

func (s *OutboxAdminService) Requeue(id string) error {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return err
	}
	eventID, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID outbox-задачи", err)
	}
	return s.repo.Requeue(eventID)
}

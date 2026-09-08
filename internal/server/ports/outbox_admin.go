package ports

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type OutboxAdminStore interface {
	Stats() (models.OutboxStats, error)
	GetFailed(int) ([]models.FailedOutboxEvent, error)
	Requeue(uuid.UUID) error
}

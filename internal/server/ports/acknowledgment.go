package ports

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AcknowledgmentConfirmationOutboxStore interface {
	MarkConfirmedWithEffects(uuid.UUID, uuid.UUID, models.AcknowledgmentConfirmationEffects) error
}

type AcknowledgmentViewedOutboxStore interface {
	MarkViewedWithOutbox(uuid.UUID, uuid.UUID, []models.OutboxEvent) error
}

type AcknowledgmentDeleteOutboxStore interface {
	DeleteWithOutbox(uuid.UUID, []models.OutboxEvent) error
}

type AcknowledgmentCreateOutboxStore interface {
	CreateWithOutbox(*models.Acknowledgment, []models.OutboxEvent) error
}

type AcknowledgmentPendingBulkStore interface {
	GetPendingForUsers([]uuid.UUID) (map[uuid.UUID][]models.Acknowledgment, error)
}

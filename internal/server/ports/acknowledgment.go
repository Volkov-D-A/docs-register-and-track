package ports

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

// AcknowledgmentStore requires atomic effects and bulk pending queries.
type AcknowledgmentStore interface {
	AcknowledgmentReader
	CreateWithOutbox(*models.Acknowledgment, []models.OutboxEvent) error
	MarkViewedWithOutbox(uuid.UUID, uuid.UUID, []models.OutboxEvent) error
	MarkConfirmedWithEffects(uuid.UUID, uuid.UUID, models.AcknowledgmentConfirmationEffects) error
	DeleteWithOutbox(uuid.UUID, []models.OutboxEvent) error
	GetPendingForUsers([]uuid.UUID) (map[uuid.UUID][]models.Acknowledgment, error)
}

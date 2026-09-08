package ports

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AdministrativeOrderAcknowledgmentOutboxStore interface {
	MarkAcknowledgmentPersonWithOutbox(uuid.UUID, uuid.UUID, []models.OutboxEvent) (*models.AdministrativeOrderAcknowledgmentPerson, error)
}

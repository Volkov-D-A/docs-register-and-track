package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type LinkOutboxStore interface {
	CreateWithOutbox(ctx context.Context, link *models.DocumentLink, effects []models.OutboxEvent) error
	DeleteWithOutbox(ctx context.Context, id uuid.UUID, effects []models.OutboxEvent) error
	CreateAndCancelOrderWithOutbox(ctx context.Context, link *models.DocumentLink, effects []models.OutboxEvent) error
}

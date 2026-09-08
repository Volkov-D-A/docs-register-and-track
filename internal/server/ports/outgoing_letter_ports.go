package ports

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type OutgoingDocumentOutboxStore interface {
	UpdateWithOutbox(models.UpdateOutgoingDocRequest, []models.OutboxEvent) (*models.OutgoingDocument, error)
}

type OutgoingDocumentJournalStore interface {
	CreateWithJournal(models.CreateOutgoingDocRequest, string, string) (*models.OutgoingDocument, error)
}

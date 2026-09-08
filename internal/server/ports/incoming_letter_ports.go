package ports

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type IncomingDocumentOutboxStore interface {
	UpdateWithOutbox(models.UpdateIncomingDocRequest, []models.OutboxEvent) (*models.IncomingDocument, error)
}

type IncomingDocumentJournalStore interface {
	CreateWithJournal(models.CreateIncomingDocRequest, string, string) (*models.IncomingDocument, error)
}

package ports

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AdministrativeOrderOutboxStore interface {
	UpdateWithOutbox(models.UpdateAdministrativeOrderDocRequest, []models.OutboxEvent) (*models.AdministrativeOrderDocument, error)
}

type AdministrativeOrderJournalStore interface {
	CreateWithJournal(models.CreateAdministrativeOrderDocRequest, string, string) (*models.AdministrativeOrderDocument, error)
}

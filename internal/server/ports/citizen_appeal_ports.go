package ports

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type CitizenAppealOutboxStore interface {
	UpdateWithOutbox(models.UpdateCitizenAppealDocRequest, []models.OutboxEvent) (*models.CitizenAppealDocument, error)
}

type CitizenAppealJournalStore interface {
	CreateWithJournal(models.CreateCitizenAppealDocRequest, string, string) (*models.CitizenAppealDocument, error)
}

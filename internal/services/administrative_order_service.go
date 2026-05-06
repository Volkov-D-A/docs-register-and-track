package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// AdministrativeOrderService предоставляет дополнительные операции по приказам.
type AdministrativeOrderService struct {
	repo    AdministrativeOrderDocStore
	auth    *AuthService
	access  *DocumentAccessService
	journal *JournalService
}

// NewAdministrativeOrderService создает сервис приказов.
func NewAdministrativeOrderService(
	repo AdministrativeOrderDocStore,
	auth *AuthService,
	access *DocumentAccessService,
	journal *JournalService,
) *AdministrativeOrderService {
	return &AdministrativeOrderService{
		repo:    repo,
		auth:    auth,
		access:  access,
		journal: journal,
	}
}

// MarkAcknowledged проставляет отметку ознакомления для строки листа приказа.
func (s *AdministrativeOrderService) MarkAcknowledged(personIDStr string) (*dto.AdministrativeOrderAcknowledgmentPerson, error) {
	personID, err := uuid.Parse(personIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid acknowledgment person ID: %w", err)
	}

	person, err := s.repo.GetAcknowledgmentPersonByID(personID)
	if err != nil {
		return nil, err
	}
	if person == nil {
		return nil, models.NewBadRequest("строка ознакомления не найдена")
	}
	if err := s.access.RequireDocumentAction(person.DocumentID, "update"); err != nil {
		return nil, err
	}

	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, models.ErrUnauthorized
	}

	updated, err := s.repo.MarkAcknowledgmentPerson(personID, userID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, models.NewBadRequest("строка ознакомления не найдена")
	}

	s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
		DocumentID: updated.DocumentID,
		UserID:     userID,
		Action:     "ORDER_ACKNOWLEDGE",
		Details:    fmt.Sprintf("Ознакомлен: %s", updated.FullName),
	})

	mapped := dto.MapAdministrativeOrderAcknowledgmentPeople([]models.AdministrativeOrderAcknowledgmentPerson{*updated})
	if len(mapped) == 0 {
		return nil, nil
	}
	return &mapped[0], nil
}

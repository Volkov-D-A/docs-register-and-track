package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// AdministrativeOrderService предоставляет дополнительные операции по приказам.
type AdministrativeOrderService struct {
	repo   AdministrativeOrderDocStore
	auth   DocumentAccessPrincipal
	access *DocumentAccessService
	server serverclient.AdministrativeOrderAcknowledgmentClient
}
type administrativeOrderAcknowledgmentOutboxStore interface {
	MarkAcknowledgmentPersonWithOutbox(uuid.UUID, uuid.UUID, []models.OutboxEvent) (*models.AdministrativeOrderAcknowledgmentPerson, error)
}

// NewAdministrativeOrderService создает сервис приказов.
func NewAdministrativeOrderService(
	repo AdministrativeOrderDocStore,
	auth DocumentAccessPrincipal,
	access *DocumentAccessService,
) *AdministrativeOrderService {
	return &AdministrativeOrderService{
		repo:   repo,
		auth:   auth,
		access: access,
	}
}

// NewAdministrativeOrderServiceWithClient creates the desktop adapter for
// server-owned administrative-order acknowledgment commands.
func NewAdministrativeOrderServiceWithClient(client serverclient.AdministrativeOrderAcknowledgmentClient) *AdministrativeOrderService {
	return &AdministrativeOrderService{server: client}
}

// MarkAcknowledged проставляет отметку ознакомления для строки листа приказа.
func (s *AdministrativeOrderService) MarkAcknowledged(personIDStr string) (*dto.AdministrativeOrderAcknowledgmentPerson, error) {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.MarkAdministrativeOrderAcknowledged(ctx, personIDStr)
	}
	personID, err := uuid.Parse(personIDStr)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID строки ознакомления", err)
	}

	person, err := s.repo.GetAcknowledgmentPersonByID(personID)
	if err != nil {
		return nil, err
	}
	if person == nil {
		return nil, models.NewNotFound("строка ознакомления не найдена")
	}
	if err := s.access.RequireDocumentAction(person.DocumentID, "update"); err != nil {
		return nil, err
	}

	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, models.ErrUnauthorized
	}

	store, ok := s.repo.(administrativeOrderAcknowledgmentOutboxStore)
	if !ok {
		return nil, fmt.Errorf("administrative order store must support atomic outbox operations")
	}
	event, buildErr := NewJournalOutboxEvent("administrative-order:"+person.DocumentID.String()+":acknowledge:"+personID.String(), models.CreateJournalEntryRequest{DocumentID: person.DocumentID, UserID: userID, Action: "ORDER_ACKNOWLEDGE", Details: fmt.Sprintf("Ознакомлен: %s", person.FullName)})
	if buildErr != nil {
		return nil, buildErr
	}
	updated, err := store.MarkAcknowledgmentPersonWithOutbox(personID, userID, []models.OutboxEvent{event})
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, models.NewNotFound("строка ознакомления не найдена")
	}

	mapped := dto.MapAdministrativeOrderAcknowledgmentPeople([]models.AdministrativeOrderAcknowledgmentPerson{*updated})
	if len(mapped) == 0 {
		return nil, nil
	}
	return &mapped[0], nil
}

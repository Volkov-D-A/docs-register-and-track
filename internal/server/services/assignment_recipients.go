package services

import (
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	servereffects "github.com/Volkov-D-A/docs-register-and-track/internal/server/effects"
	"github.com/google/uuid"
	"time"
)

func (s *AssignmentService) createRecipientTask(documentID, deadline string, userIds []string) (*dto.Assignment, error) {
	var deadlineTime *time.Time
	if deadline != "" {
		parsed, err := time.Parse("2006-01-02", deadline)
		if err != nil {
			return nil, models.NewBadRequest("неверный формат срока ознакомления")
		}
		deadlineTime = &parsed
	}
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	if err := s.access.RequireDocumentAction(docUUID, "assign"); err != nil {
		return nil, err
	}
	doc, err := s.access.RequireExists(docUUID)
	if err != nil {
		return nil, err
	}

	creatorUUID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}

	ack := &models.Assignment{
		Type: models.AssignmentTypeAcknowledgment, Status: "new", Deadline: deadlineTime,
		ID:           uuid.New(),
		DocumentID:   docUUID,
		DocumentKind: string(doc.Kind),
		CreatorID:    creatorUUID,
		CreatedAt:    time.Now(),
	}

	seenUsers := make(map[uuid.UUID]struct{}, len(userIds))
	for _, uidStr := range userIds {
		uUUID, err := uuid.Parse(uidStr)
		if err != nil || uUUID == uuid.Nil {
			return nil, models.NewBadRequest("неверный ID сотрудника для ознакомления")
		}
		if _, exists := seenUsers[uUUID]; exists {
			continue
		}
		seenUsers[uUUID] = struct{}{}
		ack.Users = append(ack.Users, models.AssignmentRecipient{
			ID:           uuid.New(),
			AssignmentID: ack.ID,
			UserID:       uUUID,
			CreatedAt:    time.Now(),
		})
	}

	if len(ack.Users) == 0 {
		return nil, models.NewBadRequest("не выбраны пользователи для ознакомления")
	}
	recipientIDs := make([]uuid.UUID, len(ack.Users))
	for i, user := range ack.Users {
		recipientIDs[i] = user.UserID
	}
	eligibleIDs, err := s.userRepo.GetEligibleRecipientIDs(recipientIDs)
	if err != nil {
		return nil, err
	}
	for _, recipientID := range recipientIDs {
		if _, ok := eligibleIDs[recipientID]; !ok {
			return nil, models.NewBadRequest("выбранный сотрудник недоступен для ознакомления")
		}
	}

	effects := make([]models.OutboxEvent, 0, len(ack.Users)+1)
	journal, buildErr := servereffects.NewJournalOutboxEvent("assignment:"+ack.ID.String()+":created:journal", models.CreateJournalEntryRequest{DocumentID: docUUID, UserID: creatorUUID, Action: "ASSIGNMENT_CREATE", Details: "Отправлен на ознакомление"})
	if buildErr != nil {
		return nil, buildErr
	}
	effects = append(effects, journal)
	for _, user := range ack.Users {
		request := models.CreateUserEventRequest{RecipientUserID: user.UserID, DocumentID: docUUID, DocumentKind: string(doc.Kind), DocumentNumber: doc.RegistrationNumber, EntityType: models.UserEventEntityAssignment, EventType: models.UserEventAssignmentCreated, Title: "Новое ознакомление", Message: "Вам направлен документ на ознакомление"}
		event, buildErr := servereffects.NewUserEventOutboxEvent("assignment:"+ack.ID.String()+":created:"+user.UserID.String(), request)
		if buildErr != nil {
			return nil, buildErr
		}
		effects = append(effects, event)
	}
	err = s.repo.CreateRecipientTaskWithOutbox(ack, effects)
	if err != nil {
		return nil, err
	}
	return dto.MapAssignment(ack), nil
}

func (s *AssignmentService) confirmAcknowledgment(ackID string) error {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return err
	}
	ackUUID, err := uuid.Parse(ackID)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID строки ознакомления", err)
	}
	ack, err := s.repo.GetByID(ackUUID)
	if err != nil {
		return err
	}
	if ack == nil || ack.Type != models.AssignmentTypeAcknowledgment {
		return models.ErrForbidden
	}
	actorID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}
	_, subjects, err := s.currentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return err
	}
	userUUID := actorID
subjectLoop:
	for _, subject := range subjects {
		for _, recipient := range ack.Users {
			if recipient.UserID.String() == subject && recipient.ConfirmedAt == nil {
				userUUID = recipient.UserID
				break subjectLoop
			}
		}
	}

	doc, _ := s.access.GetDocument(ack.DocumentID)
	documentNumber := ""
	if doc != nil {
		documentNumber = doc.RegistrationNumber
	}
	err = s.repo.ConfirmRecipientWithEffects(ackUUID, userUUID, models.AssignmentConfirmationEffects{ActorID: actorID, UserEvents: s.acknowledgmentConfirmedEventRequests(ack, documentNumber)})
	if errors.Is(err, models.ErrAlreadyConfirmed) {
		return nil
	}
	return err
}

func (s *AssignmentService) acknowledgmentConfirmedEventRequests(ack *models.Assignment, documentNumber string) []models.CreateUserEventRequest {
	if ack == nil {
		return nil
	}

	excluded := servereffects.EventActorExcluded(s.auth)
	requests := make([]models.CreateUserEventRequest, 0)
	recipients := AppendUniqueUserID(nil, ack.CreatorID)
	controlRecipients, err := s.access.CollectUserIDsWithDocumentAction(s.userRepo, ack.DocumentKind, "assign", excluded)
	if err == nil {
		for _, recipientID := range controlRecipients {
			recipients = AppendUniqueUserID(recipients, recipientID)
		}
	}

	for _, recipientID := range recipients {
		if _, skip := excluded[recipientID]; skip && recipientID != ack.CreatorID {
			continue
		}
		requests = append(requests, models.CreateUserEventRequest{
			RecipientUserID: recipientID,
			DocumentID:      ack.DocumentID,
			DocumentKind:    ack.DocumentKind,
			DocumentNumber:  documentNumber,
			EntityType:      models.UserEventEntityAssignment,
			EventType:       models.UserEventAssignmentAcknowledged,
			Title:           "Ознакомление подтверждено",
			Message:         "Пользователь подтвердил ознакомление с документом",
		})
	}
	return requests
}

func (s *AssignmentService) deleteRecipientTask(id string) error {
	ackUUID, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID строки ознакомления", err)
	}

	ack, err := s.repo.GetByID(ackUUID)
	if err != nil {
		return err
	}
	if ack == nil {
		return nil
	}
	if err := s.access.RequireDocumentAction(ack.DocumentID, "assign"); err != nil {
		return err
	}

	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}
	if ack.CreatorID != currentUserID {
		return models.ErrForbidden
	}
	var previousReaders []uuid.UUID
	for _, recipient := range ack.Users {
		previousReaders = append(previousReaders, recipient.UserID)
	}

	event, buildErr := servereffects.NewJournalOutboxEvent("assignment:"+ackUUID.String()+":deleted:journal", models.CreateJournalEntryRequest{DocumentID: ack.DocumentID, UserID: currentUserID, PreviousReaderIDs: previousReaders, Action: "ASSIGNMENT_DELETE", Details: "Ознакомление удалено"})
	if buildErr != nil {
		return buildErr
	}
	return s.repo.DeleteWithOutbox(ackUUID, []models.OutboxEvent{event})
}

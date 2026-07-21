package services

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// AcknowledgmentService предоставляет бизнес-логику для работы с задачами на ознакомление.
type AcknowledgmentService struct {
	repo          AcknowledgmentStore
	userRepo      UserStore
	auth          *AuthService
	access        *DocumentAccessService
	events        *UserEventService
	substitutions UserSubstitutionStore
}

type acknowledgmentConfirmationOutboxStore interface {
	MarkConfirmedWithEffects(uuid.UUID, uuid.UUID, models.AcknowledgmentConfirmationEffects) error
}
type acknowledgmentViewedOutboxStore interface {
	MarkViewedWithOutbox(uuid.UUID, uuid.UUID, []models.OutboxEvent) error
}
type acknowledgmentDeleteOutboxStore interface {
	DeleteWithOutbox(uuid.UUID, []models.OutboxEvent) error
}
type acknowledgmentCreateOutboxStore interface {
	CreateWithOutbox(*models.Acknowledgment, []models.OutboxEvent) error
}
type acknowledgmentPendingBulkStore interface {
	GetPendingForUsers([]uuid.UUID) (map[uuid.UUID][]models.Acknowledgment, error)
}

var errAcknowledgmentOutboxStoreRequired = errors.New("acknowledgment store must support atomic outbox operations")

// NewAcknowledgmentService создает новый экземпляр AcknowledgmentService.
func NewAcknowledgmentService(
	repo AcknowledgmentStore,
	userRepo UserStore,
	auth *AuthService,
	access *DocumentAccessService,
	events ...*UserEventService,
) *AcknowledgmentService {
	s := &AcknowledgmentService{
		repo:     repo,
		userRepo: userRepo,
		auth:     auth,
		access:   access,
	}
	if len(events) > 0 {
		s.events = events[0]
	}
	return s
}

// SetSubstitutionStore подключает источник активных замещений.
func (s *AcknowledgmentService) SetSubstitutionStore(store UserSubstitutionStore) {
	s.substitutions = store
}

func (s *AcknowledgmentService) currentUserAndSubstitutionSubjectIDs() ([]uuid.UUID, error) {
	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	ids := []uuid.UUID{currentUserID}
	if s.substitutions == nil {
		return ids, nil
	}
	principalIDs, err := s.substitutions.GetActivePrincipalIDs(currentUserID)
	if err != nil {
		return nil, err
	}
	seen := map[uuid.UUID]struct{}{currentUserID: {}}
	for _, principalID := range principalIDs {
		if principalID == uuid.Nil {
			continue
		}
		if _, ok := seen[principalID]; ok {
			continue
		}
		seen[principalID] = struct{}{}
		ids = append(ids, principalID)
	}
	return ids, nil
}

func acknowledgmentListContainsUser(acknowledgments []models.Acknowledgment, ackID uuid.UUID) bool {
	for _, ack := range acknowledgments {
		if ack.ID == ackID {
			return true
		}
	}
	return false
}

func (s *AcknowledgmentService) pendingForSubjects(subjectIDs []uuid.UUID) (map[uuid.UUID][]models.Acknowledgment, error) {
	if bulkStore, ok := s.repo.(acknowledgmentPendingBulkStore); ok {
		return bulkStore.GetPendingForUsers(subjectIDs)
	}

	result := make(map[uuid.UUID][]models.Acknowledgment, len(subjectIDs))
	for _, subjectID := range subjectIDs {
		pending, err := s.repo.GetPendingForUser(subjectID)
		if err != nil {
			return nil, err
		}
		result[subjectID] = pending
	}
	return result, nil
}

func (s *AcknowledgmentService) resolveAcknowledgmentSubjectUserID(ackID uuid.UUID) (uuid.UUID, error) {
	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return uuid.Nil, err
	}
	if s.substitutions == nil {
		return currentUserID, nil
	}
	principalIDs, err := s.substitutions.GetActivePrincipalIDs(currentUserID)
	if err != nil {
		return uuid.Nil, err
	}
	pendingBySubject, err := s.pendingForSubjects(principalIDs)
	if err != nil {
		return uuid.Nil, err
	}
	for _, subjectID := range principalIDs {
		if acknowledgmentListContainsUser(pendingBySubject[subjectID], ackID) {
			return subjectID, nil
		}
	}
	return currentUserID, nil
}

// Create создает новую задачу на ознакомление для указанных пользователей.
func (s *AcknowledgmentService) Create(
	documentID string,
	content string,
	userIds []string,
) (*dto.Acknowledgment, error) {
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	if err := s.access.RequireDocumentAction(docUUID, "acknowledge"); err != nil {
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

	ack := &models.Acknowledgment{
		ID:           uuid.New(),
		DocumentID:   docUUID,
		DocumentKind: string(doc.Kind),
		CreatorID:    creatorUUID,
		Content:      content,
		CreatedAt:    time.Now(),
	}

	for _, uidStr := range userIds {
		uUUID, err := uuid.Parse(uidStr)
		if err != nil {
			continue // пропускаем невалидные ID
		}
		ack.Users = append(ack.Users, models.AcknowledgmentUser{
			ID:               uuid.New(),
			AcknowledgmentID: ack.ID,
			UserID:           uUUID,
			CreatedAt:        time.Now(),
		})
	}

	if len(ack.Users) == 0 {
		return nil, models.NewBadRequest("не выбраны пользователи для ознакомления")
	}

	store, ok := s.repo.(acknowledgmentCreateOutboxStore)
	if !ok {
		return nil, errAcknowledgmentOutboxStoreRequired
	}
	effects := make([]models.OutboxEvent, 0, len(ack.Users)+1)
	journal, buildErr := NewJournalOutboxEvent("ack:"+ack.ID.String()+":created:journal", models.CreateJournalEntryRequest{DocumentID: docUUID, UserID: creatorUUID, Action: "ACK_CREATE", Details: "Отправлен на ознакомление"})
	if buildErr != nil {
		return nil, buildErr
	}
	effects = append(effects, journal)
	for _, user := range ack.Users {
		request := models.CreateUserEventRequest{RecipientUserID: user.UserID, ActorUserID: &creatorUUID, DocumentID: docUUID, DocumentKind: string(doc.Kind), DocumentNumber: doc.RegistrationNumber, EntityType: models.UserEventEntityAcknowledgment, EntityID: ack.ID, EventType: models.UserEventAcknowledgmentCreated, Title: "Новое ознакомление", Message: "Вам направлен документ на ознакомление", Metadata: userEventMetadata(map[string]string{"status": "pending"})}
		event, buildErr := NewUserEventOutboxEvent("ack:"+ack.ID.String()+":created:"+user.UserID.String(), request)
		if buildErr != nil {
			return nil, buildErr
		}
		effects = append(effects, event)
	}
	err = store.CreateWithOutbox(ack, effects)
	if err != nil {
		return nil, err
	}
	return dto.MapAcknowledgment(ack), nil
}

// GetList возвращает список задач на ознакомление для конкретного документа.
func (s *AcknowledgmentService) GetList(documentID string) ([]dto.Acknowledgment, error) {
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	if err := s.access.RequireDocumentAction(docUUID, "acknowledge"); err != nil {
		return nil, err
	}
	res, err := s.repo.GetByDocumentID(docUUID)
	return dto.MapAcknowledgments(res), err
}

// GetPendingForCurrentUser возвращает список невыполненных задач на ознакомление для текущего авторизованного пользователя.
func (s *AcknowledgmentService) GetPendingForCurrentUser() ([]dto.Acknowledgment, error) {
	if err := s.access.RequireDomainRead(); err != nil {
		return nil, err
	}
	subjectIDs, err := s.currentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return nil, err
	}
	pendingBySubject, err := s.pendingForSubjects(subjectIDs)
	if err != nil {
		return nil, err
	}
	result := make([]models.Acknowledgment, 0)
	seen := make(map[uuid.UUID]struct{})
	for _, subjectID := range subjectIDs {
		for _, ack := range pendingBySubject[subjectID] {
			if _, ok := seen[ack.ID]; ok {
				continue
			}
			seen[ack.ID] = struct{}{}
			result = append(result, ack)
		}
	}
	return dto.MapAcknowledgments(result), nil
}

// GetCurrentUserPendingByDocument возвращает ожидающие подтверждения ознакомления текущего пользователя по документу.
func (s *AcknowledgmentService) GetCurrentUserPendingByDocument(documentID string) ([]dto.Acknowledgment, error) {
	if err := s.access.RequireDomainRead(); err != nil {
		return nil, err
	}
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	subjectIDs, err := s.currentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return nil, err
	}

	pendingBySubject, err := s.pendingForSubjects(subjectIDs)
	if err != nil {
		return nil, err
	}
	filtered := make([]models.Acknowledgment, 0)
	seen := make(map[uuid.UUID]struct{})
	for _, subjectID := range subjectIDs {
		for _, ack := range pendingBySubject[subjectID] {
			if ack.DocumentID != docUUID {
				continue
			}
			if _, ok := seen[ack.ID]; ok {
				continue
			}
			seen[ack.ID] = struct{}{}
			filtered = append(filtered, ack)
		}
	}
	return dto.MapAcknowledgments(filtered), nil
}

// GetAllActive возвращает список всех активных (не завершенных) задач на ознакомление в системе.
// Доступно только делопроизводителям.
func (s *AcknowledgmentService) GetAllActive() ([]dto.Acknowledgment, error) {
	allowedKinds, err := s.access.GetDocumentKindsWithAction("acknowledge")
	if err != nil {
		return nil, err
	}
	if len(allowedKinds) == 0 {
		return nil, models.ErrForbidden
	}
	res, err := s.repo.GetAllActive(models.AcknowledgmentFilter{
		AllowedDocumentKinds: documentKindCodes(allowedKinds),
	})
	if err != nil {
		return nil, err
	}

	documentIDs := make([]uuid.UUID, 0, len(res))
	for _, ack := range res {
		documentIDs = append(documentIDs, ack.DocumentID)
	}
	readableDocuments, err := s.access.ResolveReadableDocuments(documentIDs)
	if err != nil {
		return nil, err
	}

	filtered := make([]models.Acknowledgment, 0, len(res))
	for _, ack := range res {
		if _, ok := readableDocuments[ack.DocumentID]; !ok {
			continue
		}
		users, err := s.repo.GetUsersByAcknowledgmentID(ack.ID)
		if err != nil {
			return nil, err
		}
		ack.Users = users
		filtered = append(filtered, ack)
	}

	return dto.MapAcknowledgments(filtered), nil
}

// MarkViewed отмечает задачу на ознакомление как просмотренную текущим пользователем.
func (s *AcknowledgmentService) MarkViewed(ackID string) error {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return err
	}
	ackUUID, err := uuid.Parse(ackID)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID строки ознакомления", err)
	}
	userUUID, err := s.resolveAcknowledgmentSubjectUserID(ackUUID)
	if err != nil {
		return err
	}

	ack, err := s.repo.GetByID(ackUUID)
	if err != nil {
		return err
	}
	if ack == nil {
		return models.ErrForbidden
	}
	store, ok := s.repo.(acknowledgmentViewedOutboxStore)
	if !ok {
		return errAcknowledgmentOutboxStoreRequired
	}
	event, buildErr := NewJournalOutboxEvent("ack:"+ackUUID.String()+":viewed:"+userUUID.String()+":journal", models.CreateJournalEntryRequest{DocumentID: ack.DocumentID, UserID: userUUID, Action: "ACK_VIEW", Details: "Документ просмотрен в рамках ознакомления"})
	if buildErr != nil {
		return buildErr
	}
	return store.MarkViewedWithOutbox(ackUUID, userUUID, []models.OutboxEvent{event})
}

// MarkConfirmed отмечает задачу на ознакомление как выполненную (подтвержденную) текущим пользователем.
func (s *AcknowledgmentService) MarkConfirmed(ackID string) error {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return err
	}
	ackUUID, err := uuid.Parse(ackID)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID строки ознакомления", err)
	}
	userUUID, err := s.resolveAcknowledgmentSubjectUserID(ackUUID)
	if err != nil {
		return err
	}

	store, ok := s.repo.(acknowledgmentConfirmationOutboxStore)
	if !ok {
		return errAcknowledgmentOutboxStoreRequired
	}
	ack, getErr := s.repo.GetByID(ackUUID)
	if getErr != nil {
		return getErr
	}
	if ack == nil {
		return models.ErrForbidden
	}
	doc, _ := s.access.GetDocument(ack.DocumentID)
	documentNumber := ""
	if doc != nil {
		documentNumber = doc.RegistrationNumber
	}
	err = store.MarkConfirmedWithEffects(ackUUID, userUUID, models.AcknowledgmentConfirmationEffects{UserEvents: s.acknowledgmentConfirmedEventRequests(ack, documentNumber, &userUUID)})
	if errors.Is(err, models.ErrAlreadyConfirmed) {
		return nil
	}
	return err
}

func (s *AcknowledgmentService) acknowledgmentConfirmedEventRequests(ack *models.Acknowledgment, documentNumber string, actorID *uuid.UUID) []models.CreateUserEventRequest {
	if ack == nil {
		return nil
	}

	excluded := eventActorExcluded(s.auth)
	requests := make([]models.CreateUserEventRequest, 0)
	recipients := appendUniqueUserID(nil, ack.CreatorID)
	controlRecipients, err := collectUserIDsWithDocumentAction(s.userRepo, s.access, ack.DocumentKind, "acknowledge", excluded)
	if err == nil {
		for _, recipientID := range controlRecipients {
			recipients = appendUniqueUserID(recipients, recipientID)
		}
	}

	for _, recipientID := range recipients {
		if _, skip := excluded[recipientID]; skip && recipientID != ack.CreatorID {
			continue
		}
		requests = append(requests, models.CreateUserEventRequest{
			RecipientUserID: recipientID,
			ActorUserID:     actorID,
			DocumentID:      ack.DocumentID,
			DocumentKind:    ack.DocumentKind,
			DocumentNumber:  documentNumber,
			EntityType:      models.UserEventEntityAcknowledgment,
			EntityID:        ack.ID,
			EventType:       models.UserEventAcknowledgmentConfirmed,
			Title:           "Ознакомление подтверждено",
			Message:         "Пользователь подтвердил ознакомление с документом",
			Metadata: userEventMetadata(map[string]string{
				"status": "completed",
			}),
		})
	}
	return requests
}

// Delete удаляет задачу на ознакомление по её ID.
func (s *AcknowledgmentService) Delete(id string) error {
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
	if err := s.access.RequireDocumentAction(ack.DocumentID, "acknowledge"); err != nil {
		return err
	}

	store, ok := s.repo.(acknowledgmentDeleteOutboxStore)
	if !ok {
		return errAcknowledgmentOutboxStoreRequired
	}
	currentUserID, _ := s.auth.GetCurrentUserUUID()
	event, buildErr := NewJournalOutboxEvent("ack:"+ackUUID.String()+":deleted:journal", models.CreateJournalEntryRequest{DocumentID: ack.DocumentID, UserID: currentUserID, Action: "ACK_DELETE", Details: "Ознакомление удалено"})
	if buildErr != nil {
		return buildErr
	}
	return store.DeleteWithOutbox(ackUUID, []models.OutboxEvent{event})
}

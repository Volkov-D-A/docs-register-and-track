package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// AssignmentService предоставляет бизнес-логику для работы с поручениями.
type AssignmentService struct {
	repo     AssignmentStore
	userRepo UserStore // Для валидации
	auth     *AuthService
	journal  *JournalService
	access   *DocumentAccessService
	events   *UserEventService
}

// NewAssignmentService создает новый экземпляр AssignmentService.
func NewAssignmentService(
	repo AssignmentStore,
	userRepo UserStore,
	auth *AuthService,
	journal *JournalService,
	access *DocumentAccessService,
	events ...*UserEventService,
) *AssignmentService {
	s := &AssignmentService{
		repo:     repo,
		userRepo: userRepo,
		auth:     auth,
		journal:  journal,
		access:   access,
	}
	if len(events) > 0 {
		s.events = events[0]
	}
	return s
}

// Create — создание поручения
func (s *AssignmentService) Create(
	documentID string,
	executorID string,
	content string,
	deadline string,
	coExecutorIDs []string,
) (*dto.Assignment, error) {
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

	execUUID, err := uuid.Parse(executorID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID исполнителя", err)
	}

	var deadlineTime *time.Time
	if deadline != "" {
		t, err := time.Parse("2006-01-02", deadline)
		if err != nil {
			return nil, models.NewBadRequestWrapped("неверный формат срока исполнения", err)
		}
		deadlineTime = &t
	}

	res, err := s.repo.Create(docUUID, execUUID, content, deadlineTime, coExecutorIDs)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: docUUID,
			UserID:     currentUserID,
			Action:     "ASSIGNMENT_CREATE",
			Details:    fmt.Sprintf("Создано поручение для %s", doc.Kind),
		})
		s.createAssignmentCreatedEvents(res, eventActorID(s.auth))
	}
	return dto.MapAssignment(res), err
}

// Update — редактирование (админ)
func (s *AssignmentService) Update(
	id string,
	executorID string,
	content string,
	deadline string,
	coExecutorIDs []string,
) (*dto.Assignment, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
	}

	// Проверка прав доступа
	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, models.NewNotFound("поручение не найдено")
	}
	if err := s.access.RequireDocumentAction(existing.DocumentID, "assign"); err != nil {
		return nil, err
	}

	// Проверка прав
	// Редактировать могут админ и делопроизводитель
	// Завершенные поручения редактировать нельзя
	if existing.Status == "finished" {
		return nil, models.NewConflict("нельзя редактировать завершённое поручение")
	}

	execUUID, err := uuid.Parse(executorID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID исполнителя", err)
	}

	var deadlineTime *time.Time
	if deadline != "" {
		t, err := time.Parse("2006-01-02", deadline)
		if err != nil {
			return nil, models.NewBadRequestWrapped("неверный формат срока исполнения", err)
		}
		deadlineTime = &t
	}

	res, err := s.repo.Update(uid, execUUID, content, deadlineTime, existing.Status, existing.Report, existing.CompletedAt, coExecutorIDs)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: existing.DocumentID,
			UserID:     currentUserID,
			Action:     "ASSIGNMENT_UPDATE",
			Details:    "Поручение отредактировано",
		})
		s.createAssignmentUpdatedEvents(res, eventActorID(s.auth))
	}
	return dto.MapAssignment(res), err
}

// UpdateStatus — изменение статуса (исполнитель или админ)
func (s *AssignmentService) UpdateStatus(id, status, report string) (*dto.Assignment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
	}

	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, models.NewNotFound("поручение не найдено")
	}

	currentUserID := s.auth.GetCurrentUserID()
	isExecutor := existing.ExecutorID.String() == currentUserID
	canManageAssignment := s.access.RequireDocumentAction(existing.DocumentID, "assign") == nil
	allowed := false
	if canManageAssignment && existing.Status == "completed" && (status == "finished" || status == "returned") {
		allowed = true
	}
	if isExecutor && (status == "in_progress" || status == "completed") {
		allowed = true
	}

	if !allowed {
		return nil, models.NewForbidden(fmt.Sprintf("недостаточно прав для установки статуса %s", status))
	}

	report = strings.TrimSpace(report)
	switch status {
	case "completed":
		if report == "" {
			return nil, models.NewBadRequest("отчет об исполнении обязателен")
		}
	case "returned":
		if report == "" {
			return nil, models.NewBadRequest("причина возврата обязательна")
		}
	case "finished":
		if report == "" {
			report = existing.Report
		}
	}

	// Вычисление даты завершения
	var completedAt *time.Time
	switch status {
	case "completed":
		now := time.Now()
		completedAt = &now
	case "new", "in_progress":
		// Сброс даты завершения при возврате в активный статус
		completedAt = nil
	case "finished":
		completedAt = existing.CompletedAt
	default:
		completedAt = nil
	}

	res, err := s.repo.Update(uid, existing.ExecutorID, existing.Content, existing.Deadline, status, report, completedAt, existing.CoExecutorIDs)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: existing.DocumentID,
			UserID:     currentUserID,
			Action:     "ASSIGNMENT_STATUS",
			Details:    fmt.Sprintf("Статус поручения изменен на %s", status),
		})
		s.createAssignmentStatusEvents(res, existing.Status, status, report, eventActorID(s.auth))
	}
	return dto.MapAssignment(res), err
}

// GetByID возвращает поручение по его ID.
func (s *AssignmentService) GetByID(id string) (*dto.Assignment, error) {
	if err := s.access.RequireDomainRead(); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID поручения", err)
	}
	res, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, models.NewNotFound("поручение не найдено")
	}
	if err := s.access.RequireDocumentAction(res.DocumentID, "assign"); err != nil && !isAssignmentAccessibleToExecutor(s.auth.GetCurrentUserID(), res) {
		return nil, models.ErrForbidden
	}
	return dto.MapAssignment(res), nil
}

// GetList возвращает список поручений с учетом фильтрации.
func (s *AssignmentService) GetList(filter models.AssignmentFilter) (*dto.PagedResult[dto.Assignment], error) {
	if err := s.access.RequireDomainRead(); err != nil {
		return nil, err
	}
	// Значения по умолчанию
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.DocumentID != "" {
		docUUID, err := uuid.Parse(filter.DocumentID)
		if err != nil {
			return nil, models.NewBadRequestWrapped("неверный ID документа", err)
		}
		if err := s.access.RequireDocumentAction(docUUID, "assign"); err != nil {
			filter.AccessibleByUserID = s.auth.GetCurrentUserID()
		}
	}

	assignableKinds, err := s.access.GetDocumentKindsWithAction("assign")
	if err != nil {
		return nil, err
	}
	if len(assignableKinds) == 0 {
		filter.ExecutorID = s.auth.GetCurrentUserID()
	} else if len(assignableKinds) < len(models.AllDocumentKindSpecs()) {
		filter.AllowedDocumentKinds = documentKindCodes(assignableKinds)
		filter.AccessibleByUserID = s.auth.GetCurrentUserID()
	}

	res, err := s.repo.GetList(filter)
	if err != nil {
		return nil, err
	}
	return &dto.PagedResult[dto.Assignment]{
		Items:      dto.MapAssignments(res.Items),
		TotalCount: res.TotalCount,
		Page:       res.Page,
		PageSize:   res.PageSize,
	}, nil
}

func documentKindCodes(kinds []models.DocumentKind) []string {
	codes := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		codes = append(codes, string(kind))
	}
	return codes
}

func (s *AssignmentService) createAssignmentCreatedEvents(assignment *models.Assignment, actorID *uuid.UUID) {
	if s.events == nil || assignment == nil {
		return
	}

	recipients := appendUniqueUserID(nil, assignment.ExecutorID)
	for _, coExecutorID := range assignment.CoExecutorIDs {
		uid, err := uuid.Parse(coExecutorID)
		if err == nil {
			recipients = appendUniqueUserID(recipients, uid)
		}
	}

	for _, recipientID := range recipients {
		createUserEventIfEnabled(s.events, models.CreateUserEventRequest{
			RecipientUserID: recipientID,
			ActorUserID:     actorID,
			DocumentID:      assignment.DocumentID,
			DocumentKind:    assignment.DocumentKind,
			DocumentNumber:  assignment.DocumentNumber,
			EntityType:      models.UserEventEntityAssignment,
			EntityID:        assignment.ID,
			EventType:       models.UserEventAssignmentCreated,
			Title:           "Новое поручение",
			Message:         fmt.Sprintf("Вам назначено поручение по документу %s", documentNumberLabel(assignment.DocumentNumber)),
			Metadata: userEventMetadata(map[string]string{
				"status": assignment.Status,
			}),
		})
	}
}

func (s *AssignmentService) createAssignmentUpdatedEvents(assignment *models.Assignment, actorID *uuid.UUID) {
	if s.events == nil || assignment == nil {
		return
	}

	recipients := appendUniqueUserID(nil, assignment.ExecutorID)
	for _, coExecutorID := range assignment.CoExecutorIDs {
		uid, err := uuid.Parse(coExecutorID)
		if err == nil {
			recipients = appendUniqueUserID(recipients, uid)
		}
	}

	for _, recipientID := range recipients {
		createUserEventIfEnabled(s.events, models.CreateUserEventRequest{
			RecipientUserID: recipientID,
			ActorUserID:     actorID,
			DocumentID:      assignment.DocumentID,
			DocumentKind:    assignment.DocumentKind,
			DocumentNumber:  assignment.DocumentNumber,
			EntityType:      models.UserEventEntityAssignment,
			EntityID:        assignment.ID,
			EventType:       models.UserEventAssignmentUpdated,
			Title:           "Поручение изменено",
			Message:         fmt.Sprintf("Изменено поручение по документу %s", documentNumberLabel(assignment.DocumentNumber)),
			Metadata: userEventMetadata(map[string]string{
				"status": assignment.Status,
			}),
		})
	}
}

func (s *AssignmentService) createAssignmentStatusEvents(
	assignment *models.Assignment,
	oldStatus string,
	newStatus string,
	report string,
	actorID *uuid.UUID,
) {
	if s.events == nil || assignment == nil || oldStatus == newStatus {
		return
	}

	switch newStatus {
	case "completed":
		s.createAssignmentControlEvents(assignment, actorID, models.UserEventAssignmentCompleted, "Поручение ожидает приемки", fmt.Sprintf("Исполнитель отправил поручение по документу %s на приемку", documentNumberLabel(assignment.DocumentNumber)), report)
	case "finished":
		s.createAssignmentExecutorEvent(assignment, actorID, models.UserEventAssignmentFinished, "Поручение принято", fmt.Sprintf("Исполненное поручение по документу %s принято", documentNumberLabel(assignment.DocumentNumber)), report)
	case "returned":
		s.createAssignmentExecutorEvent(assignment, actorID, models.UserEventAssignmentReturned, "Поручение отклонено", fmt.Sprintf("Поручение по документу %s возвращено на доработку", documentNumberLabel(assignment.DocumentNumber)), report)
	}
}

func (s *AssignmentService) createAssignmentExecutorEvent(
	assignment *models.Assignment,
	actorID *uuid.UUID,
	eventType string,
	title string,
	message string,
	report string,
) {
	recipients := appendUniqueUserID(nil, assignment.ExecutorID)
	for _, coExecutorID := range assignment.CoExecutorIDs {
		uid, err := uuid.Parse(coExecutorID)
		if err == nil {
			recipients = appendUniqueUserID(recipients, uid)
		}
	}

	for _, recipientID := range recipients {
		createUserEventIfEnabled(s.events, models.CreateUserEventRequest{
			RecipientUserID: recipientID,
			ActorUserID:     actorID,
			DocumentID:      assignment.DocumentID,
			DocumentKind:    assignment.DocumentKind,
			DocumentNumber:  assignment.DocumentNumber,
			EntityType:      models.UserEventEntityAssignment,
			EntityID:        assignment.ID,
			EventType:       eventType,
			Title:           title,
			Message:         message,
			Metadata: userEventMetadata(map[string]string{
				"status": assignment.Status,
				"report": report,
			}),
		})
	}
}

func (s *AssignmentService) createAssignmentControlEvents(
	assignment *models.Assignment,
	actorID *uuid.UUID,
	eventType string,
	title string,
	message string,
	report string,
) {
	excluded := eventActorExcluded(s.auth)
	excluded[assignment.ExecutorID] = struct{}{}
	for _, coExecutorID := range assignment.CoExecutorIDs {
		uid, err := uuid.Parse(coExecutorID)
		if err == nil {
			excluded[uid] = struct{}{}
		}
	}

	recipients, err := collectUserIDsWithDocumentAction(s.userRepo, s.access, assignment.DocumentKind, "assign", excluded)
	if err != nil {
		return
	}
	for _, recipientID := range recipients {
		createUserEventIfEnabled(s.events, models.CreateUserEventRequest{
			RecipientUserID: recipientID,
			ActorUserID:     actorID,
			DocumentID:      assignment.DocumentID,
			DocumentKind:    assignment.DocumentKind,
			DocumentNumber:  assignment.DocumentNumber,
			EntityType:      models.UserEventEntityAssignment,
			EntityID:        assignment.ID,
			EventType:       eventType,
			Title:           title,
			Message:         message,
			Metadata: userEventMetadata(map[string]string{
				"status": assignment.Status,
				"report": report,
			}),
		})
	}
}

// Delete удаляет поручение по его ID (только для незавершенных, если не админ).
func (s *AssignmentService) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID поручения", err)
	}

	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return err
	}
	if existing == nil {
		return models.NewNotFound("поручение не найдено")
	}
	if err := s.access.RequireDocumentAction(existing.DocumentID, "assign"); err != nil {
		return err
	}

	// Завершенные поручения удалять нельзя
	if existing.Status == "finished" {
		return models.NewConflict("нельзя удалить завершённое поручение")
	}

	err = s.repo.Delete(uid)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: existing.DocumentID,
			UserID:     currentUserID,
			Action:     "ASSIGNMENT_DELETE",
			Details:    "Поручение удалено",
		})
	}
	return err
}

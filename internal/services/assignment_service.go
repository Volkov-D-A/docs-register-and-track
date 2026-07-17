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
	repo          AssignmentStore
	userRepo      UserStore
	auth          *AuthService
	journal       *JournalService
	access        *DocumentAccessService
	events        *UserEventService
	substitutions UserSubstitutionStore
}

type assignmentOutboxStore interface {
	CreateWithOutbox(id, documentID, executorID uuid.UUID, content string, deadline *time.Time, coExecutorIDs []string, effects []models.OutboxEvent) (*models.Assignment, error)
	UpdateWithOutbox(id, executorID uuid.UUID, content string, deadline *time.Time, status, report string, completedAt *time.Time, coExecutorIDs []string, effects []models.OutboxEvent) (*models.Assignment, error)
	DeleteWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error
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

// SetSubstitutionStore подключает источник активных замещений.
func (s *AssignmentService) SetSubstitutionStore(store UserSubstitutionStore) {
	s.substitutions = store
}

func (s *AssignmentService) currentUserAndSubstitutionSubjectIDs() ([]uuid.UUID, []string, error) {
	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, nil, err
	}
	ids := []uuid.UUID{currentUserID}
	if s.substitutions != nil {
		principalIDs, err := s.substitutions.GetActivePrincipalIDs(currentUserID)
		if err != nil {
			return nil, nil, err
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
	}
	return ids, uuidStrings(ids), nil
}

func (s *AssignmentService) assignmentActorAccess(existing *models.Assignment, currentUserID uuid.UUID) (bool, bool, bool, error) {
	isExecutor := existing.ExecutorID == currentUserID
	isSubstituteExecutor := false
	if !isExecutor && s.substitutions != nil {
		ok, err := s.substitutions.IsActiveSubstitute(currentUserID, existing.ExecutorID)
		if err != nil {
			return false, false, false, err
		}
		isSubstituteExecutor = ok
	}
	canManageAssignment := s.access.RequireDocumentAction(existing.DocumentID, "assign") == nil
	return isExecutor, isSubstituteExecutor, canManageAssignment, nil
}

type assignmentStatusUpdate struct {
	report      string
	completedAt *time.Time
}

var executorAssignmentTransitions = map[string]map[string]struct{}{
	"new": {
		"in_progress": {},
		"completed":   {},
	},
	"in_progress": {
		"completed": {},
	},
	"returned": {
		"in_progress": {},
		"completed":   {},
	},
}

var managerAssignmentTransitions = map[string]map[string]struct{}{
	"completed": {
		"finished": {},
		"returned": {},
	},
}

func isAssignmentTransitionAllowed(transitions map[string]map[string]struct{}, currentStatus, targetStatus string) bool {
	targets, ok := transitions[currentStatus]
	if !ok {
		return false
	}
	_, ok = targets[targetStatus]
	return ok
}

func resolveAssignmentStatusUpdate(existing *models.Assignment, status, report string, canManageAssignment, canActAsExecutor bool) (*assignmentStatusUpdate, error) {
	if existing == nil {
		return nil, models.NewNotFound("поручение не найдено")
	}

	allowed := (canManageAssignment && isAssignmentTransitionAllowed(managerAssignmentTransitions, existing.Status, status)) ||
		(canActAsExecutor && isAssignmentTransitionAllowed(executorAssignmentTransitions, existing.Status, status))
	if !allowed {
		if canManageAssignment || canActAsExecutor {
			return nil, models.NewConflict(fmt.Sprintf("недопустимый переход статуса поручения: %s → %s", existing.Status, status))
		}
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

	var completedAt *time.Time
	switch status {
	case "completed":
		now := time.Now()
		completedAt = &now
	case "finished":
		completedAt = existing.CompletedAt
	}

	return &assignmentStatusUpdate{report: report, completedAt: completedAt}, nil
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

	repo, ok := s.repo.(assignmentOutboxStore)
	if !ok {
		return nil, fmt.Errorf("assignment store must support atomic outbox operations")
	}
	var res *models.Assignment
	{
		assignmentID := uuid.New()
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		journalRequest := models.CreateJournalEntryRequest{DocumentID: docUUID, UserID: currentUserID, Action: "ASSIGNMENT_CREATE", Details: fmt.Sprintf("Создано поручение для %s", doc.Kind)}
		effects := make([]models.OutboxEvent, 0, 1+len(coExecutorIDs))
		journalEvent, buildErr := NewJournalOutboxEvent(assignmentOutboxKey(assignmentID, "created", "", nil, "journal"), journalRequest)
		if buildErr != nil {
			return nil, buildErr
		}
		effects = append(effects, journalEvent)
		assignment := &models.Assignment{ID: assignmentID, DocumentID: docUUID, DocumentKind: string(doc.Kind), DocumentNumber: doc.RegistrationNumber, ExecutorID: execUUID, CoExecutorIDs: coExecutorIDs, Status: "new"}
		for _, recipientID := range assignmentExecutorRecipientIDs(assignment) {
			request := models.CreateUserEventRequest{RecipientUserID: recipientID, ActorUserID: eventActorID(s.auth), DocumentID: docUUID, DocumentKind: string(doc.Kind), DocumentNumber: doc.RegistrationNumber, EntityType: models.UserEventEntityAssignment, EntityID: assignmentID, EventType: models.UserEventAssignmentCreated, Title: "Новое поручение", Message: fmt.Sprintf("Вам назначено поручение по документу %s", documentNumberLabel(doc.RegistrationNumber)), Metadata: userEventMetadata(map[string]string{"status": "new"})}
			event, buildErr := NewUserEventOutboxEvent(assignmentOutboxKey(assignmentID, "created", "", &recipientID, "user_event"), request)
			if buildErr != nil {
				return nil, buildErr
			}
			effects = append(effects, event)
		}
		res, err = repo.CreateWithOutbox(assignmentID, docUUID, execUUID, content, deadlineTime, coExecutorIDs, effects)
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

	repo, ok := s.repo.(assignmentOutboxStore)
	if !ok {
		return nil, fmt.Errorf("assignment store must support atomic outbox operations")
	}
	var res *models.Assignment
	{
		revision := time.Now().UTC().Format(time.RFC3339Nano)
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		journal, buildErr := NewJournalOutboxEvent(assignmentOutboxKey(uid, "updated", revision, nil, "journal"), models.CreateJournalEntryRequest{DocumentID: existing.DocumentID, UserID: currentUserID, Action: "ASSIGNMENT_UPDATE", Details: "Поручение отредактировано"})
		if buildErr != nil {
			return nil, buildErr
		}
		effects := []models.OutboxEvent{journal}
		updated := &models.Assignment{ID: uid, DocumentID: existing.DocumentID, DocumentKind: existing.DocumentKind, DocumentNumber: existing.DocumentNumber, ExecutorID: execUUID, CoExecutorIDs: coExecutorIDs, Status: existing.Status, UpdatedAt: time.Now()}
		for _, recipient := range assignmentExecutorRecipientIDs(updated) {
			request := models.CreateUserEventRequest{RecipientUserID: recipient, ActorUserID: eventActorID(s.auth), DocumentID: updated.DocumentID, DocumentKind: updated.DocumentKind, DocumentNumber: updated.DocumentNumber, EntityType: models.UserEventEntityAssignment, EntityID: updated.ID, EventType: models.UserEventAssignmentUpdated, Title: "Поручение изменено", Message: fmt.Sprintf("Изменено поручение по документу %s", documentNumberLabel(updated.DocumentNumber)), Metadata: userEventMetadata(map[string]string{"status": updated.Status})}
			event, buildErr := NewUserEventOutboxEvent(assignmentOutboxKey(uid, "updated", revision, &recipient, "user_event"), request)
			if buildErr != nil {
				return nil, buildErr
			}
			effects = append(effects, event)
		}
		res, err = repo.UpdateWithOutbox(uid, execUUID, content, deadlineTime, existing.Status, existing.Report, existing.CompletedAt, coExecutorIDs, effects)
	}
	return dto.MapAssignment(res), err
}

// UpdateStatus — изменение статуса (исполнитель или админ)
func (s *AssignmentService) UpdateStatus(id, status, report string) (*dto.Assignment, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
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

	currentUserUUID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	isExecutor, isSubstituteExecutor, canManageAssignment, err := s.assignmentActorAccess(existing, currentUserUUID)
	if err != nil {
		return nil, err
	}
	statusUpdate, err := resolveAssignmentStatusUpdate(existing, status, report, canManageAssignment, isExecutor || isSubstituteExecutor)
	if err != nil {
		return nil, err
	}

	repo, ok := s.repo.(assignmentOutboxStore)
	if !ok {
		return nil, fmt.Errorf("assignment store must support atomic outbox operations")
	}
	var res *models.Assignment
	{
		revision := time.Now().UTC().Format(time.RFC3339Nano)
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		journal, buildErr := NewJournalOutboxEvent(assignmentOutboxKey(uid, "status:"+status, revision, nil, "journal"), models.CreateJournalEntryRequest{DocumentID: existing.DocumentID, UserID: currentUserID, Action: "ASSIGNMENT_STATUS", Details: fmt.Sprintf("Статус поручения изменен на %s", status)})
		if buildErr != nil {
			return nil, buildErr
		}
		effects := []models.OutboxEvent{journal}
		updated := *existing
		updated.Status = status
		updated.Report = statusUpdate.report
		updated.CompletedAt = statusUpdate.completedAt
		updated.UpdatedAt = time.Now()
		var recipients []uuid.UUID
		var eventType, title, message string
		switch status {
		case "completed":
			if s.events != nil {
				recipients, _ = collectUserIDsWithDocumentAction(s.userRepo, s.access, updated.DocumentKind, "assign", nil)
			}
			eventType, title, message = models.UserEventAssignmentCompleted, "Поручение ожидает приемки", fmt.Sprintf("Исполнитель отправил поручение по документу %s на приемку", documentNumberLabel(updated.DocumentNumber))
		case "finished":
			recipients = assignmentExecutorRecipientIDs(&updated)
			eventType, title, message = models.UserEventAssignmentFinished, "Поручение принято", fmt.Sprintf("Исполненное поручение по документу %s принято", documentNumberLabel(updated.DocumentNumber))
		case "returned":
			recipients = assignmentExecutorRecipientIDs(&updated)
			eventType, title, message = models.UserEventAssignmentReturned, "Поручение отклонено", fmt.Sprintf("Поручение по документу %s возвращено на доработку", documentNumberLabel(updated.DocumentNumber))
		}
		if s.events != nil {
			for _, recipient := range recipients {
				request := models.CreateUserEventRequest{RecipientUserID: recipient, ActorUserID: eventActorID(s.auth), DocumentID: updated.DocumentID, DocumentKind: updated.DocumentKind, DocumentNumber: updated.DocumentNumber, EntityType: models.UserEventEntityAssignment, EntityID: updated.ID, EventType: eventType, Title: title, Message: message, Metadata: userEventMetadata(map[string]string{"status": status, "report": statusUpdate.report})}
				event, buildErr := NewUserEventOutboxEvent(assignmentOutboxKey(uid, eventType, revision, &recipient, "user_event"), request)
				if buildErr != nil {
					return nil, buildErr
				}
				effects = append(effects, event)
			}
		}
		res, err = repo.UpdateWithOutbox(uid, existing.ExecutorID, existing.Content, existing.Deadline, status, statusUpdate.report, statusUpdate.completedAt, existing.CoExecutorIDs, effects)
	}
	mapped := dto.MapAssignment(res)
	if mapped != nil {
		mapped.CanAct = isExecutor || isSubstituteExecutor
	}
	return mapped, err
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
	if err := s.access.RequireDocumentAction(res.DocumentID, "assign"); err != nil {
		_, subjectIDs, subjectsErr := s.currentUserAndSubstitutionSubjectIDs()
		if subjectsErr != nil {
			return nil, subjectsErr
		}
		if !isAssignmentAccessibleToAnyExecutor(subjectIDs, res) {
			return nil, models.ErrForbidden
		}
	}
	mapped := dto.MapAssignment(res)
	if mapped != nil {
		_, subjectIDs, subjectsErr := s.currentUserAndSubstitutionSubjectIDs()
		if subjectsErr != nil {
			return nil, subjectsErr
		}
		mapped.CanAct = isAssignmentExecutorInSubjects(subjectIDs, res)
	}
	return mapped, nil
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
			_, subjectIDs, subjectsErr := s.currentUserAndSubstitutionSubjectIDs()
			if subjectsErr != nil {
				return nil, subjectsErr
			}
			filter.AccessibleByUserID = subjectIDs[0]
			if len(subjectIDs) == 1 {
				filter.ExecutorID = subjectIDs[0]
			} else {
				filter.ExecutorID = ""
				filter.AccessibleByUserIDs = subjectIDs
			}
		}
	}

	assignableKinds, err := s.access.GetDocumentKindsWithAction("assign")
	if err != nil {
		return nil, err
	}
	_, subjectIDs, err := s.currentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return nil, err
	}
	if len(assignableKinds) == 0 {
		if len(subjectIDs) == 1 {
			filter.ExecutorID = subjectIDs[0]
		} else {
			filter.ExecutorID = ""
			filter.AccessibleByUserID = subjectIDs[0]
			filter.AccessibleByUserIDs = subjectIDs
		}
	} else if len(assignableKinds) < len(models.AllDocumentKindSpecs()) {
		filter.AllowedDocumentKinds = documentKindCodes(assignableKinds)
		filter.AccessibleByUserID = subjectIDs[0]
		if len(subjectIDs) > 1 {
			filter.AccessibleByUserIDs = subjectIDs
		}
	}

	res, err := s.repo.GetList(filter)
	if err != nil {
		return nil, err
	}
	items := dto.MapAssignments(res.Items)
	markAssignmentsCanAct(items, subjectIDs, res.Items)
	return &dto.PagedResult[dto.Assignment]{
		Items:      items,
		TotalCount: res.TotalCount,
		Page:       res.Page,
		PageSize:   res.PageSize,
	}, nil
}

func markAssignmentsCanAct(items []dto.Assignment, subjectIDs []string, assignments []models.Assignment) {
	for i := range items {
		if i < len(assignments) {
			items[i].CanAct = isAssignmentExecutorInSubjects(subjectIDs, &assignments[i])
		}
	}
}

func isAssignmentExecutorInSubjects(subjectIDs []string, assignment *models.Assignment) bool {
	if assignment == nil {
		return false
	}
	executorID := assignment.ExecutorID.String()
	for _, subjectID := range subjectIDs {
		if subjectID == executorID {
			return true
		}
	}
	return false
}

func documentKindCodes(kinds []models.DocumentKind) []string {
	codes := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		codes = append(codes, string(kind))
	}
	return codes
}

func assignmentExecutorRecipientIDs(assignment *models.Assignment) []uuid.UUID {
	if assignment == nil {
		return nil
	}
	recipients := appendUniqueUserID(nil, assignment.ExecutorID)
	for _, coExecutorID := range assignment.CoExecutorIDs {
		uid, err := uuid.Parse(coExecutorID)
		if err == nil {
			recipients = appendUniqueUserID(recipients, uid)
		}
	}
	return recipients
}

// assignmentOutboxKey separates the business transition, its persisted
// revision, recipient and effect kind. Thus a retry repeats one effect, while
// a later transition creates a distinct task.
func assignmentOutboxKey(id uuid.UUID, transition, revision string, recipient *uuid.UUID, effect string) string {
	parts := []string{"assignment", id.String(), transition}
	if revision != "" {
		parts = append(parts, revision)
	}
	if recipient != nil {
		parts = append(parts, recipient.String())
	}
	parts = append(parts, effect)
	return strings.Join(parts, ":")
}

func assignmentRevision(assignment *models.Assignment) string {
	if assignment == nil {
		return ""
	}
	return assignment.UpdatedAt.UTC().Format(time.RFC3339Nano)
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

	repo, ok := s.repo.(assignmentOutboxStore)
	if !ok {
		return fmt.Errorf("assignment store must support atomic outbox operations")
	}
	currentUserID, _ := s.auth.GetCurrentUserUUID()
	event, buildErr := NewJournalOutboxEvent(assignmentOutboxKey(uid, "deleted", "", nil, "journal"), models.CreateJournalEntryRequest{DocumentID: existing.DocumentID, UserID: currentUserID, Action: "ASSIGNMENT_DELETE", Details: "Поручение удалено"})
	if buildErr != nil {
		return buildErr
	}
	err = repo.DeleteWithOutbox(uid, []models.OutboxEvent{event})
	return err
}

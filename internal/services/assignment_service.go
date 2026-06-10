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

func resolveAssignmentStatusUpdate(existing *models.Assignment, status, report string, canManageAssignment, canActAsExecutor bool) (*assignmentStatusUpdate, error) {
	allowed := false
	if canManageAssignment && existing.Status == "completed" && (status == "finished" || status == "returned") {
		allowed = true
	}
	if canActAsExecutor && (status == "in_progress" || status == "completed") {
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

	res, err := s.repo.Update(uid, existing.ExecutorID, existing.Content, existing.Deadline, status, statusUpdate.report, statusUpdate.completedAt, existing.CoExecutorIDs)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(nil, models.CreateJournalEntryRequest{
			DocumentID: existing.DocumentID,
			UserID:     currentUserID,
			Action:     "ASSIGNMENT_STATUS",
			Details:    fmt.Sprintf("Статус поручения изменен на %s", status),
		})
		s.createAssignmentStatusEvents(res, existing.Status, status, statusUpdate.report, eventActorID(s.auth))
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

func (s *AssignmentService) createAssignmentEvent(
	assignment *models.Assignment,
	recipientID uuid.UUID,
	actorID *uuid.UUID,
	eventType string,
	title string,
	message string,
	metadata map[string]string,
) {
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
		Metadata:        userEventMetadata(metadata),
	})
}

func (s *AssignmentService) createAssignmentCreatedEvents(assignment *models.Assignment, actorID *uuid.UUID) {
	if s.events == nil || assignment == nil {
		return
	}
	for _, recipientID := range assignmentExecutorRecipientIDs(assignment) {
		s.createAssignmentEvent(
			assignment,
			recipientID,
			actorID,
			models.UserEventAssignmentCreated,
			"Новое поручение",
			fmt.Sprintf("Вам назначено поручение по документу %s", documentNumberLabel(assignment.DocumentNumber)),
			map[string]string{
				"status": assignment.Status,
			},
		)
	}
}

func (s *AssignmentService) createAssignmentUpdatedEvents(assignment *models.Assignment, actorID *uuid.UUID) {
	if s.events == nil || assignment == nil {
		return
	}
	for _, recipientID := range assignmentExecutorRecipientIDs(assignment) {
		s.createAssignmentEvent(
			assignment,
			recipientID,
			actorID,
			models.UserEventAssignmentUpdated,
			"Поручение изменено",
			fmt.Sprintf("Изменено поручение по документу %s", documentNumberLabel(assignment.DocumentNumber)),
			map[string]string{
				"status": assignment.Status,
			},
		)
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
	for _, recipientID := range assignmentExecutorRecipientIDs(assignment) {
		s.createAssignmentEvent(
			assignment,
			recipientID,
			actorID,
			eventType,
			title,
			message,
			map[string]string{
				"status": assignment.Status,
				"report": report,
			},
		)
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
	recipients, err := collectUserIDsWithDocumentAction(s.userRepo, s.access, assignment.DocumentKind, "assign", nil)
	if err != nil {
		return
	}
	for _, recipientID := range recipients {
		s.createAssignmentEvent(
			assignment,
			recipientID,
			actorID,
			eventType,
			title,
			message,
			map[string]string{
				"status": assignment.Status,
				"report": report,
			},
		)
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

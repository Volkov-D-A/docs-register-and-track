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
	access        *DocumentAccessService
	events        *UserEventService
	substitutions UserSubstitutionStore
}

type assignmentOutboxStore interface {
	CreateWithOutbox(id, documentID, executorID uuid.UUID, content string, deadline *time.Time, coExecutorIDs []string, effects []models.OutboxEvent) (*models.Assignment, error)
	UpdateWithOutbox(id, executorID uuid.UUID, content string, deadline *time.Time, status, report string, completedAt *time.Time, coExecutorIDs []string, effects []models.OutboxEvent) (*models.Assignment, error)
	DeleteWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error
}

type assignmentSeriesStore interface {
	CreateSeriesWithFirstAssignment(seriesID, assignmentID, documentID, executorID, createdBy uuid.UUID, content string, firstDeadline time.Time, intervalUnit string, intervalValue int, dayRule string, dayOfMonth int, coExecutorIDs []string, effects []models.OutboxEvent) (*models.AssignmentSeries, error)
	GetAssignmentSeries(id uuid.UUID) (*models.AssignmentSeries, error)
	GetAssignmentSeriesByAssignment(id uuid.UUID) (*models.AssignmentSeries, error)
	UpdateAssignmentSeries(id, executorID uuid.UUID, content, intervalUnit string, intervalValue int, dayRule string, dayOfMonth int, coExecutorIDs []string, effects []models.OutboxEvent) (*models.AssignmentSeries, error)
	CancelAssignmentSeries(id, actorID uuid.UUID, effects []models.OutboxEvent) error
	FinishSeriesIterationWithNext(currentID, seriesID, nextID uuid.UUID, expectedSeriesUpdatedAt time.Time, report string, completedAt *time.Time, nextDeadline time.Time, nextIteration int, executorID uuid.UUID, content string, coExecutorIDs []string, currentEffects, nextEffects []models.OutboxEvent) (*models.Assignment, error)
	GetAssignmentSeriesHistory(seriesID uuid.UUID) ([]models.Assignment, error)
}

// NewAssignmentService создает новый экземпляр AssignmentService.
func NewAssignmentService(
	repo AssignmentStore,
	userRepo UserStore,
	auth *AuthService,
	access *DocumentAccessService,
	events ...*UserEventService,
) *AssignmentService {
	s := &AssignmentService{
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

func validateAssignmentSeriesRequest(request models.AssignmentSeriesRequest, requireFirstDeadline bool) (uuid.UUID, string, string, int, int, error) {
	executorID, err := uuid.Parse(request.ExecutorID)
	if err != nil {
		return uuid.Nil, "", "", 0, 0, models.NewBadRequestWrapped("неверный ID исполнителя", err)
	}
	content := strings.TrimSpace(request.Content)
	if content == "" {
		return uuid.Nil, "", "", 0, 0, models.NewBadRequest("текст поручения обязателен")
	}
	if request.IntervalUnit != "day" && request.IntervalUnit != "week" && request.IntervalUnit != "month" && request.IntervalUnit != "year" {
		return uuid.Nil, "", "", 0, 0, models.NewBadRequest("неверная единица интервала серии")
	}
	if request.IntervalValue < 1 || request.IntervalValue > 3650 {
		return uuid.Nil, "", "", 0, 0, models.NewBadRequest("значение интервала серии должно быть от 1 до 3650")
	}
	day := request.DayOfMonth
	dayRule := request.DayRule
	if request.IntervalUnit == "day" || request.IntervalUnit == "week" {
		dayRule, day = "same_day", 0
	} else if dayRule != "fixed" && dayRule != "last_day" {
		return uuid.Nil, "", "", 0, 0, models.NewBadRequest("неверное правило календарного дня")
	}
	if dayRule == "fixed" && (day < 1 || day > 31) {
		return uuid.Nil, "", "", 0, 0, models.NewBadRequest("день месяца должен быть от 1 до 31")
	}
	if dayRule == "last_day" || dayRule == "same_day" {
		day = 0
	}
	if requireFirstDeadline && request.FirstDeadline == "" {
		return uuid.Nil, "", "", 0, 0, models.NewBadRequest("укажите первый плановый срок")
	}
	return executorID, content, dayRule, request.IntervalValue, day, nil
}

func dateMatchesSeriesRule(value time.Time, dayRule string, dayOfMonth int) bool {
	if dayRule == "last_day" {
		return value.Day() == time.Date(value.Year(), value.Month()+1, 0, 0, 0, 0, 0, value.Location()).Day()
	}
	expected := dayOfMonth
	last := time.Date(value.Year(), value.Month()+1, 0, 0, 0, 0, 0, value.Location()).Day()
	if expected > last {
		expected = last
	}
	return value.Day() == expected
}

func nextSeriesDeadline(current time.Time, intervalUnit string, intervalValue int, dayRule string, dayOfMonth int) time.Time {
	if intervalUnit == "day" {
		return current.AddDate(0, 0, intervalValue)
	}
	if intervalUnit == "week" {
		return current.AddDate(0, 0, intervalValue*7)
	}
	months := intervalValue
	if intervalUnit == "year" {
		months = intervalValue * 12
	}
	targetMonth := time.Date(current.Year(), current.Month()+time.Month(months), 1, 0, 0, 0, 0, current.Location())
	last := time.Date(targetMonth.Year(), targetMonth.Month()+1, 0, 0, 0, 0, 0, current.Location()).Day()
	day := dayOfMonth
	if dayRule == "last_day" || day > last {
		day = last
	}
	return time.Date(targetMonth.Year(), targetMonth.Month(), day, 0, 0, 0, 0, current.Location())
}

// CreateSeries creates the recurring template and its first ordinary iteration.
func (s *AssignmentService) CreateSeries(request models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	documentID, err := uuid.Parse(request.DocumentID)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	if err = s.access.RequireDocumentAction(documentID, "assign"); err != nil {
		return nil, err
	}
	doc, err := s.access.RequireExists(documentID)
	if err != nil {
		return nil, err
	}
	executorID, content, dayRule, intervalValue, dayOfMonth, err := validateAssignmentSeriesRequest(request, true)
	if err != nil {
		return nil, err
	}
	firstDeadline, err := time.Parse("2006-01-02", request.FirstDeadline)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный формат первого срока", err)
	}
	if dayRule != "same_day" && !dateMatchesSeriesRule(firstDeadline, dayRule, dayOfMonth) {
		return nil, models.NewBadRequest("первый срок не соответствует выбранному правилу расписания")
	}
	repo, ok := s.repo.(assignmentSeriesStore)
	if !ok {
		return nil, fmt.Errorf("assignment store must support series operations")
	}
	seriesID, assignmentID := uuid.New(), uuid.New()
	actorID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	effects := make([]models.OutboxEvent, 0, 2+len(request.CoExecutorIDs))
	journal, err := NewJournalOutboxEvent(assignmentOutboxKey(assignmentID, "series-created", "", nil, "journal"), models.CreateJournalEntryRequest{DocumentID: documentID, UserID: actorID, Action: "ASSIGNMENT_SERIES_CREATE", Details: "Создана серия поручений и первая итерация"})
	if err != nil {
		return nil, err
	}
	effects = append(effects, journal)
	assignment := &models.Assignment{ID: assignmentID, DocumentID: documentID, DocumentKind: string(doc.Kind), DocumentNumber: doc.RegistrationNumber, ExecutorID: executorID, CoExecutorIDs: request.CoExecutorIDs, Status: "new", SeriesID: &seriesID, IterationNumber: 1}
	for _, recipientID := range assignmentExecutorRecipientIDs(assignment) {
		req := models.CreateUserEventRequest{RecipientUserID: recipientID, ActorUserID: eventActorID(s.auth), DocumentID: documentID, DocumentKind: string(doc.Kind), DocumentNumber: doc.RegistrationNumber, EntityType: models.UserEventEntityAssignment, EntityID: assignmentID, EventType: models.UserEventAssignmentCreated, Title: "Новое поручение", Message: fmt.Sprintf("Вам назначено поручение по документу %s", documentNumberLabel(doc.RegistrationNumber)), Metadata: userEventMetadata(map[string]string{"status": "new"})}
		event, buildErr := NewUserEventOutboxEvent(assignmentOutboxKey(assignmentID, "created", "", &recipientID, "user_event"), req)
		if buildErr != nil {
			return nil, buildErr
		}
		effects = append(effects, event)
	}
	result, err := repo.CreateSeriesWithFirstAssignment(seriesID, assignmentID, documentID, executorID, actorID, content, firstDeadline, request.IntervalUnit, intervalValue, dayRule, dayOfMonth, request.CoExecutorIDs, effects)
	return dto.MapAssignmentSeries(result), err
}

func (s *AssignmentService) getManagedSeries(id string) (*models.AssignmentSeries, assignmentSeriesStore, error) {
	seriesID, err := uuid.Parse(id)
	if err != nil {
		return nil, nil, models.NewBadRequestWrapped("неверный ID серии", err)
	}
	repo, ok := s.repo.(assignmentSeriesStore)
	if !ok {
		return nil, nil, fmt.Errorf("assignment store must support series operations")
	}
	series, err := repo.GetAssignmentSeries(seriesID)
	if err != nil {
		return nil, nil, err
	}
	if series == nil {
		return nil, nil, models.NewNotFound("серия поручений не найдена")
	}
	if err = s.access.RequireDocumentAction(series.DocumentID, "assign"); err != nil {
		return nil, nil, err
	}
	return series, repo, nil
}

func (s *AssignmentService) GetSeries(id string) (*dto.AssignmentSeries, error) {
	series, _, err := s.getManagedSeries(id)
	return dto.MapAssignmentSeries(series), err
}

func (s *AssignmentService) GetSeriesHistory(id string) ([]dto.Assignment, error) {
	series, repo, err := s.getManagedSeries(id)
	if err != nil {
		return nil, err
	}
	items, err := repo.GetAssignmentSeriesHistory(series.ID)
	if err != nil {
		return nil, err
	}
	return dto.MapAssignments(items), nil
}

func (s *AssignmentService) UpdateSeries(id string, request models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	series, repo, err := s.getManagedSeries(id)
	if err != nil {
		return nil, err
	}
	executorID, content, dayRule, intervalValue, dayOfMonth, err := validateAssignmentSeriesRequest(request, false)
	if err != nil {
		return nil, err
	}
	actorID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	event, err := NewJournalOutboxEvent("assignment-series:"+series.ID.String()+":"+time.Now().UTC().Format(time.RFC3339Nano)+":updated:journal", models.CreateJournalEntryRequest{DocumentID: series.DocumentID, UserID: actorID, Action: "ASSIGNMENT_SERIES_UPDATE", Details: "Параметры будущих итераций поручения изменены"})
	if err != nil {
		return nil, err
	}
	result, err := repo.UpdateAssignmentSeries(series.ID, executorID, content, request.IntervalUnit, intervalValue, dayRule, dayOfMonth, request.CoExecutorIDs, []models.OutboxEvent{event})
	return dto.MapAssignmentSeries(result), err
}

func (s *AssignmentService) CancelSeries(id string) error {
	series, repo, err := s.getManagedSeries(id)
	if err != nil {
		return err
	}
	actorID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}
	event, err := NewJournalOutboxEvent("assignment-series:"+series.ID.String()+":cancelled:journal", models.CreateJournalEntryRequest{DocumentID: series.DocumentID, UserID: actorID, Action: "ASSIGNMENT_SERIES_CANCEL", Details: "Серия поручений отменена; текущая итерация сохранена"})
	if err != nil {
		return err
	}
	return repo.CancelAssignmentSeries(series.ID, actorID, []models.OutboxEvent{event})
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
		if status == "finished" && existing.SeriesID != nil {
			seriesRepo, seriesOK := s.repo.(assignmentSeriesStore)
			if !seriesOK {
				return nil, fmt.Errorf("assignment store must support series operations")
			}
			series, seriesErr := seriesRepo.GetAssignmentSeries(*existing.SeriesID)
			if seriesErr != nil {
				return nil, seriesErr
			}
			if series == nil {
				return nil, models.NewConflict("серия поручения не найдена")
			}
			baseDeadline := existing.PlannedDeadline
			if baseDeadline == nil {
				baseDeadline = existing.Deadline
			}
			if baseDeadline == nil {
				return nil, models.NewConflict("для серийного поручения не задан плановый срок")
			}
			nextDeadline := nextSeriesDeadline(*baseDeadline, series.IntervalUnit, series.IntervalValue, series.DayRule, series.DayOfMonth)
			nextID := uuid.New()
			nextIteration := existing.IterationNumber + 1
			nextEffects := make([]models.OutboxEvent, 0, 1+len(series.CoExecutorIDs))
			nextJournal, buildErr := NewJournalOutboxEvent(assignmentOutboxKey(nextID, "series-iteration-created", "", nil, "journal"), models.CreateJournalEntryRequest{DocumentID: existing.DocumentID, UserID: currentUserID, Action: "ASSIGNMENT_SERIES_ITERATION_CREATE", Details: fmt.Sprintf("Автоматически создана итерация поручения №%d со сроком %s", nextIteration, nextDeadline.Format("02.01.2006"))})
			if buildErr != nil {
				return nil, buildErr
			}
			nextEffects = append(nextEffects, nextJournal)
			nextAssignment := &models.Assignment{ID: nextID, DocumentID: existing.DocumentID, DocumentKind: existing.DocumentKind, DocumentNumber: existing.DocumentNumber, ExecutorID: series.ExecutorID, CoExecutorIDs: series.CoExecutorIDs, Status: "new", SeriesID: &series.ID, IterationNumber: nextIteration}
			for _, recipientID := range assignmentExecutorRecipientIDs(nextAssignment) {
				request := models.CreateUserEventRequest{RecipientUserID: recipientID, ActorUserID: eventActorID(s.auth), DocumentID: existing.DocumentID, DocumentKind: existing.DocumentKind, DocumentNumber: existing.DocumentNumber, EntityType: models.UserEventEntityAssignment, EntityID: nextID, EventType: models.UserEventAssignmentCreated, Title: "Новое поручение", Message: fmt.Sprintf("Вам назначено поручение по документу %s", documentNumberLabel(existing.DocumentNumber)), Metadata: userEventMetadata(map[string]string{"status": "new"})}
				event, eventErr := NewUserEventOutboxEvent(assignmentOutboxKey(nextID, "created", "", &recipientID, "user_event"), request)
				if eventErr != nil {
					return nil, eventErr
				}
				nextEffects = append(nextEffects, event)
			}
			res, err = seriesRepo.FinishSeriesIterationWithNext(uid, series.ID, nextID, series.UpdatedAt, statusUpdate.report, statusUpdate.completedAt, nextDeadline, nextIteration, series.ExecutorID, series.Content, series.CoExecutorIDs, effects, nextEffects)
		} else {
			res, err = repo.UpdateWithOutbox(uid, existing.ExecutorID, existing.Content, existing.Deadline, status, statusUpdate.report, statusUpdate.completedAt, existing.CoExecutorIDs, effects)
		}
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
	manageErr := s.access.RequireDocumentAction(res.DocumentID, "assign")
	if res.SeriesID != nil && !res.IsSeriesCurrent && manageErr != nil {
		return nil, models.ErrForbidden
	}
	if manageErr != nil {
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
	if existing.SeriesID != nil {
		return models.NewConflict("итерацию серии нельзя удалить отдельно; отмените серию или завершите текущую итерацию")
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

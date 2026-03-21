package services

import (
	"context"
	"fmt"
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
}

// NewAssignmentService создает новый экземпляр AssignmentService.
func NewAssignmentService(
	repo AssignmentStore,
	userRepo UserStore,
	auth *AuthService,
	journal *JournalService,
) *AssignmentService {
	return &AssignmentService{
		repo:     repo,
		userRepo: userRepo,
		auth:     auth,
		journal:  journal,
	}
}

// Create — создание поручения
func (s *AssignmentService) Create(
	documentID, documentType string,
	executorID string,
	content string,
	deadline string,
	coExecutorIDs []string,
) (*dto.Assignment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}

	execUUID, err := uuid.Parse(executorID)
	if err != nil {
		return nil, fmt.Errorf("invalid executor ID: %w", err)
	}

	var deadlineTime *time.Time
	if deadline != "" {
		t, err := time.Parse("2006-01-02", deadline)
		if err != nil {
			return nil, fmt.Errorf("invalid deadline format: %w", err)
		}
		deadlineTime = &t
	}

	res, err := s.repo.Create(docUUID, documentType, execUUID, content, deadlineTime, coExecutorIDs)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   docUUID,
			DocumentType: documentType,
			UserID:       currentUserID,
			Action:       "ASSIGNMENT_CREATE",
			Details:      "Создано поручение",
		})
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
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}

	// Проверка прав доступа
	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("поручение не найдено")
	}

	// Проверка прав
	// Редактировать могут админ и делопроизводитель
	if err := s.auth.RequireAnyRole("admin", "clerk"); err != nil {
		return nil, err
	}

	// Завершенные поручения редактировать нельзя (кроме админа)
	if existing.Status == "finished" && !s.auth.HasRole("admin") {
		return nil, fmt.Errorf("нельзя редактировать завершённое поручение")
	}

	execUUID, err := uuid.Parse(executorID)
	if err != nil {
		return nil, fmt.Errorf("invalid executor ID: %w", err)
	}

	var deadlineTime *time.Time
	if deadline != "" {
		t, err := time.Parse("2006-01-02", deadline)
		if err != nil {
			return nil, fmt.Errorf("invalid deadline format: %w", err)
		}
		deadlineTime = &t
	}

	res, err := s.repo.Update(uid, execUUID, content, deadlineTime, existing.Status, existing.Report, existing.CompletedAt, coExecutorIDs)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   existing.DocumentID,
			DocumentType: existing.DocumentType,
			UserID:       currentUserID,
			Action:       "ASSIGNMENT_UPDATE",
			Details:      "Поручение отредактировано",
		})
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
		return nil, fmt.Errorf("invalid ID: %w", err)
	}

	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("поручение не найдено")
	}

	currentUserID := s.auth.GetCurrentUserID()
	isExecutor := existing.ExecutorID.String() == currentUserID

	isAdmin := s.auth.HasRole("admin")
	isClerk := s.auth.HasRole("clerk")

	allowed := false
	if isAdmin {
		allowed = true
	}
	if isClerk {
		// Делопроизводитель может завершить или вернуть поручение только из статуса "completed"
		if existing.Status == "completed" && (status == "finished" || status == "returned") {
			allowed = true
		}
	}
	if isExecutor {
		if status == "in_progress" || status == "completed" {
			allowed = true
		}
	}

	if !allowed {
		return nil, models.NewForbidden(fmt.Sprintf("недостаточно прав для установки статуса %s", status))
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
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   existing.DocumentID,
			DocumentType: existing.DocumentType,
			UserID:       currentUserID,
			Action:       "ASSIGNMENT_STATUS",
			Details:      fmt.Sprintf("Статус поручения изменен на %s", status),
		})
	}
	return dto.MapAssignment(res), err
}

// GetByID возвращает поручение по его ID.
func (s *AssignmentService) GetByID(id string) (*dto.Assignment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	res, err := s.repo.GetByID(uid)
	return dto.MapAssignment(res), err
}

// GetList возвращает список поручений с учетом фильтрации.
func (s *AssignmentService) GetList(filter models.AssignmentFilter) (*dto.PagedResult[dto.Assignment], error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	// Значения по умолчанию
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
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

// Delete удаляет поручение по его ID (только для незавершенных, если не админ).
func (s *AssignmentService) Delete(id string) error {
	if !s.auth.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}

	// Удалять могут админ и делопроизводитель
	if err := s.auth.RequireAnyRole("admin", "clerk"); err != nil {
		return err
	}

	// Завершенные поручения удалять нельзя (кроме админа)
	if existing.Status == "finished" && !s.auth.HasRole("admin") {
		return fmt.Errorf("нельзя удалить завершённое поручение")
	}

	err = s.repo.Delete(uid)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   existing.DocumentID,
			DocumentType: existing.DocumentType,
			UserID:       currentUserID,
			Action:       "ASSIGNMENT_DELETE",
			Details:      "Поручение удалено",
		})
	}
	return err
}

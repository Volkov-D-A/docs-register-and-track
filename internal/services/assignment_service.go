package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/models"
	"docflow/internal/repository"
)

type AssignmentService struct {
	ctx      context.Context
	repo     *repository.AssignmentRepository
	userRepo *repository.UserRepository // Для валидации
	auth     *AuthService
}

func NewAssignmentService(
	repo *repository.AssignmentRepository,
	userRepo *repository.UserRepository,
	auth *AuthService,
) *AssignmentService {
	return &AssignmentService{
		repo:     repo,
		userRepo: userRepo,
		auth:     auth,
	}
}

func (s *AssignmentService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// Create — создание поручения
func (s *AssignmentService) Create(
	documentID, documentType string,
	executorID string,
	content string,
	deadline string,
	coExecutorIDs []string,
) (*models.Assignment, error) {
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

	return s.repo.Create(docUUID, documentType, execUUID, content, deadlineTime, coExecutorIDs)
}

// Update — редактирование (админ)
func (s *AssignmentService) Update(
	id string,
	executorID string,
	content string,
	deadline string,
	coExecutorIDs []string,
) (*models.Assignment, error) {
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
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав")
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

	return s.repo.Update(uid, execUUID, content, deadlineTime, existing.Status, existing.Report, existing.CompletedAt, coExecutorIDs)
}

// UpdateStatus — изменение статуса (исполнитель или админ)
func (s *AssignmentService) UpdateStatus(id, status, report string) (*models.Assignment, error) {
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

	// Правила доступа
	// Админ: все статусы
	// Исполнитель: in_progress, completed

	isClerk := s.auth.HasRole("clerk")

	allowed := false
	if isAdmin {
		allowed = true
	} else if isClerk {
		// Делопроизводитель может завершить или вернуть поручение только из статуса "completed"
		if existing.Status == "completed" && (status == "finished" || status == "returned") {
			allowed = true
		}
	} else if isExecutor {
		if status == "in_progress" || status == "completed" {
			allowed = true
		}
	}

	if !allowed {
		return nil, fmt.Errorf("недостаточно прав для установки статуса %s", status)
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

	return s.repo.Update(uid, existing.ExecutorID, existing.Content, existing.Deadline, status, report, completedAt, existing.CoExecutorIDs)
}

func (s *AssignmentService) GetByID(id string) (*models.Assignment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.GetByID(uid)
}

func (s *AssignmentService) GetList(filter models.AssignmentFilter) (*models.PagedResult[models.Assignment], error) {
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
	return s.repo.GetList(filter)
}

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
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return fmt.Errorf("недостаточно прав")
	}

	// Завершенные поручения удалять нельзя (кроме админа)
	if existing.Status == "finished" && !s.auth.HasRole("admin") {
		return fmt.Errorf("нельзя удалить завершённое поручение")
	}

	return s.repo.Delete(uid)
}

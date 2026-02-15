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
	userRepo *repository.UserRepository // For validation
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

	// Check permissions
	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("assignment not found")
	}

	// Check permissions
	// Admin or Clerk can edit
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("permission denied")
	}

	// Restriction: cannot edit finished assignments (unless admin, maybe? original request said "cannot edit finished", implies strict rule)
	// Let's apply strict rule for clerks. Admins might need to fix mistakes, but let's follow "Завершенные поручения редактировать и удалять нельзя" strictly for the requested feature (clerks).
	if existing.Status == "finished" && !s.auth.HasRole("admin") {
		return nil, fmt.Errorf("cannot edit finished assignment")
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
		return nil, fmt.Errorf("assignment not found")
	}

	currentUserID := s.auth.GetCurrentUserID()
	isExecutor := existing.ExecutorID.String() == currentUserID
	isAdmin := s.auth.HasRole("admin")

	// Rules
	// Admin: all
	// Executor: in_progress, completed

	isClerk := s.auth.HasRole("clerk")

	allowed := false
	if isAdmin {
		allowed = true
	} else if isClerk {
		// Clerk can finish or return assignments ONLY if they are currently completed (executed)
		if existing.Status == "completed" && (status == "finished" || status == "returned") {
			allowed = true
		}
	} else if isExecutor {
		if status == "in_progress" || status == "completed" {
			allowed = true
		}
	}

	if !allowed {
		return nil, fmt.Errorf("permission denied to set status %s", status)
	}

	// Calculate completedAt
	var completedAt *time.Time
	switch status {
	case "completed":
		now := time.Now()
		completedAt = &now
	case "new", "in_progress":
		// Reset completion time if moved back to active
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

func (s *AssignmentService) GetList(filter models.AssignmentFilter) (*models.PagedResult, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	// Defaults
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

	// Admin or Clerk can delete
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return fmt.Errorf("permission denied")
	}

	// Restriction: cannot delete finished assignments
	if existing.Status == "finished" && !s.auth.HasRole("admin") {
		return fmt.Errorf("cannot delete finished assignment")
	}

	return s.repo.Delete(uid)
}

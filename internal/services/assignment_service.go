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

	return s.repo.Create(docUUID, documentType, execUUID, content, deadlineTime)
}

// Update — редактирование (админ)
func (s *AssignmentService) Update(
	id string,
	executorID string,
	content string,
	deadline string,
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

	// Only admin can edit assignment details
	if !s.auth.HasRole("admin") {
		return nil, fmt.Errorf("permission denied")
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

	return s.repo.Update(uid, execUUID, content, deadlineTime, existing.Status, existing.Report)
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

	allowed := false
	if isAdmin {
		allowed = true
	} else if isExecutor {
		if status == "in_progress" || status == "completed" {
			allowed = true
		}
	}

	if !allowed {
		return nil, fmt.Errorf("permission denied to set status %s", status)
	}

	return s.repo.Update(uid, existing.ExecutorID, existing.Content, existing.Deadline, status, report)
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

	if !s.auth.HasRole("admin") {
		return fmt.Errorf("permission denied")
	}

	return s.repo.Delete(uid)
}

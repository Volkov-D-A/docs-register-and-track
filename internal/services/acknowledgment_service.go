package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/models"
	"docflow/internal/repository"
)

type AcknowledgmentService struct {
	ctx      context.Context
	repo     *repository.AcknowledgmentRepository
	userRepo *repository.UserRepository
	auth     *AuthService
}

func NewAcknowledgmentService(
	repo *repository.AcknowledgmentRepository,
	userRepo *repository.UserRepository,
	auth *AuthService,
) *AcknowledgmentService {
	return &AcknowledgmentService{
		repo:     repo,
		userRepo: userRepo,
		auth:     auth,
	}
}

func (s *AcknowledgmentService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *AcknowledgmentService) Create(
	documentID, documentType string,
	content string,
	userIds []string,
) (*models.Acknowledgment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	// Permission: Clerk or Admin
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("permission denied")
	}

	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}

	creatorID := s.auth.GetCurrentUserID()
	creatorUUID, err := uuid.Parse(creatorID)
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	ack := &models.Acknowledgment{
		ID:           uuid.New(),
		DocumentID:   docUUID,
		DocumentType: documentType,
		CreatorID:    creatorUUID,
		Content:      content,
		CreatedAt:    time.Now(),
	}

	for _, uidStr := range userIds {
		uUUID, err := uuid.Parse(uidStr)
		if err != nil {
			continue // or error out
		}
		ack.Users = append(ack.Users, models.AcknowledgmentUser{
			ID:               uuid.New(),
			AcknowledgmentID: ack.ID,
			UserID:           uUUID,
			CreatedAt:        time.Now(),
		})
	}

	if len(ack.Users) == 0 {
		return nil, fmt.Errorf("no users selected for acknowledgment")
	}

	err = s.repo.Create(ack)
	if err != nil {
		return nil, err
	}

	// Prepare result with strings filled
	ack.FillIDStr()
	return ack, nil
}

func (s *AcknowledgmentService) GetList(documentID string) ([]models.Acknowledgment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	return s.repo.GetByDocumentID(docUUID)
}

func (s *AcknowledgmentService) GetPendingForCurrentUser() ([]models.Acknowledgment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	userID := s.auth.GetCurrentUserID()
	userUUID, _ := uuid.Parse(userID)
	return s.repo.GetPendingForUser(userUUID)
}

func (s *AcknowledgmentService) GetAllActive() ([]models.Acknowledgment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("permission denied")
	}
	return s.repo.GetAllActive()
}

func (s *AcknowledgmentService) MarkViewed(ackID string) error {
	if !s.auth.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	ackUUID, err := uuid.Parse(ackID)
	if err != nil {
		return fmt.Errorf("invalid acknowledgment ID: %w", err)
	}
	userID := s.auth.GetCurrentUserID()
	userUUID, _ := uuid.Parse(userID)

	return s.repo.MarkViewed(ackUUID, userUUID)
}

func (s *AcknowledgmentService) MarkConfirmed(ackID string) error {
	if !s.auth.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	ackUUID, err := uuid.Parse(ackID)
	if err != nil {
		return fmt.Errorf("invalid acknowledgment ID: %w", err)
	}
	userID := s.auth.GetCurrentUserID()
	userUUID, _ := uuid.Parse(userID)

	return s.repo.MarkConfirmed(ackUUID, userUUID)
}

func (s *AcknowledgmentService) Delete(id string) error {
	if !s.auth.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	// Only creator (clerk/admin) should delete? Or just admin?
	// Let's allow admin and clerk.
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return fmt.Errorf("permission denied")
	}

	ackUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Delete(ackUUID)
}

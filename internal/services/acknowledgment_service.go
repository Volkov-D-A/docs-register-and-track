package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/dto"
	"docflow/internal/models"
)

// AcknowledgmentService предоставляет бизнес-логику для работы с задачами на ознакомление.
type AcknowledgmentService struct {
	repo     AcknowledgmentStore
	userRepo UserStore
	auth     *AuthService
}

// NewAcknowledgmentService создает новый экземпляр AcknowledgmentService.
func NewAcknowledgmentService(
	repo AcknowledgmentStore,
	userRepo UserStore,
	auth *AuthService,
) *AcknowledgmentService {
	return &AcknowledgmentService{
		repo:     repo,
		userRepo: userRepo,
		auth:     auth,
	}
}

// Create создает новую задачу на ознакомление для указанных пользователей.
func (s *AcknowledgmentService) Create(
	documentID, documentType string,
	content string,
	userIds []string,
) (*dto.Acknowledgment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	// Проверка прав: делопроизводитель или админ
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, models.ErrForbidden
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
		return nil, fmt.Errorf("не выбраны пользователи для ознакомления")
	}

	err = s.repo.Create(ack)
	if err != nil {
		return nil, err
	}

	// Заполнение строковых ID для результата

	return dto.MapAcknowledgment(ack), nil
}

// GetList возвращает список задач на ознакомление для конкретного документа.
func (s *AcknowledgmentService) GetList(documentID string) ([]dto.Acknowledgment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	res, err := s.repo.GetByDocumentID(docUUID)
	return dto.MapAcknowledgments(res), err
}

// GetPendingForCurrentUser возвращает список невыполненных задач на ознакомление для текущего авторизованного пользователя.
func (s *AcknowledgmentService) GetPendingForCurrentUser() ([]dto.Acknowledgment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	userID := s.auth.GetCurrentUserID()
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrNotAuthenticated
	}
	res, err := s.repo.GetPendingForUser(userUUID)
	return dto.MapAcknowledgments(res), err
}

// GetAllActive возвращает список всех активных (не завершенных) задач на ознакомление в системе.
// Доступно только администраторам и делопроизводителям.
func (s *AcknowledgmentService) GetAllActive() ([]dto.Acknowledgment, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, models.ErrForbidden
	}
	res, err := s.repo.GetAllActive()
	return dto.MapAcknowledgments(res), err
}

// MarkViewed отмечает задачу на ознакомление как просмотренную текущим пользователем.
func (s *AcknowledgmentService) MarkViewed(ackID string) error {
	if !s.auth.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	ackUUID, err := uuid.Parse(ackID)
	if err != nil {
		return fmt.Errorf("invalid acknowledgment ID: %w", err)
	}
	userID := s.auth.GetCurrentUserID()
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotAuthenticated
	}

	return s.repo.MarkViewed(ackUUID, userUUID)
}

// MarkConfirmed отмечает задачу на ознакомление как выполненную (подтвержденную) текущим пользователем.
func (s *AcknowledgmentService) MarkConfirmed(ackID string) error {
	if !s.auth.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	ackUUID, err := uuid.Parse(ackID)
	if err != nil {
		return fmt.Errorf("invalid acknowledgment ID: %w", err)
	}
	userID := s.auth.GetCurrentUserID()
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotAuthenticated
	}

	return s.repo.MarkConfirmed(ackUUID, userUUID)
}

// Delete удаляет задачу на ознакомление по её ID.
func (s *AcknowledgmentService) Delete(id string) error {
	if !s.auth.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	// Удалять могут админ и делопроизводитель
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return models.ErrForbidden
	}

	ackUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Delete(ackUUID)
}

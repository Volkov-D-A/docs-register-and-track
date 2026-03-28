package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// AcknowledgmentService предоставляет бизнес-логику для работы с задачами на ознакомление.
type AcknowledgmentService struct {
	repo     AcknowledgmentStore
	userRepo UserStore
	auth     *AuthService
	journal  *JournalService
}

// NewAcknowledgmentService создает новый экземпляр AcknowledgmentService.
func NewAcknowledgmentService(
	repo AcknowledgmentStore,
	userRepo UserStore,
	auth *AuthService,
	journal *JournalService,
) *AcknowledgmentService {
	return &AcknowledgmentService{
		repo:     repo,
		userRepo: userRepo,
		auth:     auth,
		journal:  journal,
	}
}

// Create создает новую задачу на ознакомление для указанных пользователей.
func (s *AcknowledgmentService) Create(
	documentID, documentType string,
	content string,
	userIds []string,
) (*dto.Acknowledgment, error) {
	if err := requireClerkDocumentRole(s.auth); err != nil {
		return nil, err
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

	s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
		DocumentID:   docUUID,
		DocumentType: documentType,
		UserID:       creatorUUID,
		Action:       "ACK_CREATE",
		Details:      "Отправлен на ознакомление",
	})

	// Заполнение строковых ID для результата

	return dto.MapAcknowledgment(ack), nil
}

// GetList возвращает список задач на ознакомление для конкретного документа.
func (s *AcknowledgmentService) GetList(documentID string) ([]dto.Acknowledgment, error) {
	if err := requireClerkDocumentRole(s.auth); err != nil {
		return nil, err
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
	if err := s.auth.RequireAnyActiveRole("executor", "clerk"); err != nil {
		return nil, err
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
// Доступно только делопроизводителям.
func (s *AcknowledgmentService) GetAllActive() ([]dto.Acknowledgment, error) {
	if err := requireClerkDocumentRole(s.auth); err != nil {
		return nil, err
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

	err = s.repo.MarkViewed(ackUUID, userUUID)
	if err == nil {
		ack, _ := s.repo.GetByID(ackUUID)
		if ack != nil {
			s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
				DocumentID:   ack.DocumentID,
				DocumentType: ack.DocumentType,
				UserID:       userUUID,
				Action:       "ACK_VIEW",
				Details:      "Документ просмотрен в рамках ознакомления",
			})
		}
	}
	return err
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

	err = s.repo.MarkConfirmed(ackUUID, userUUID)
	if err == nil {
		ack, _ := s.repo.GetByID(ackUUID)
		if ack != nil {
			s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
				DocumentID:   ack.DocumentID,
				DocumentType: ack.DocumentType,
				UserID:       userUUID,
				Action:       "ACK_CONFIRM",
				Details:      "Ознакомление подтверждено",
			})
		}
	}
	return err
}

// Delete удаляет задачу на ознакомление по её ID.
func (s *AcknowledgmentService) Delete(id string) error {
	if err := requireClerkDocumentRole(s.auth); err != nil {
		return err
	}

	ackUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	ack, err := s.repo.GetByID(ackUUID)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ackUUID)
	if err == nil {
		currentUserID, _ := s.auth.GetCurrentUserUUID()
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID:   ack.DocumentID,
			DocumentType: ack.DocumentType,
			UserID:       currentUserID,
			Action:       "ACK_DELETE",
			Details:      "Ознакомление удалено",
		})
	}

	return err
}

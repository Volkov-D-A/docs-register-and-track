package services

import (
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// UserEventService предоставляет бизнес-логику персональных событий.
type UserEventService struct {
	repo UserEventStore
	auth *AuthService
}

// NewUserEventService создает новый экземпляр UserEventService.
func NewUserEventService(repo UserEventStore, auth *AuthService) *UserEventService {
	return &UserEventService{repo: repo, auth: auth}
}

// create создает событие. Используется внутренними сервисами.
func (s *UserEventService) create(req models.CreateUserEventRequest) (*dto.UserEvent, error) {
	if req.RecipientUserID == uuid.Nil {
		return nil, models.NewBadRequest("не указан получатель события")
	}
	if req.DocumentID == uuid.Nil {
		return nil, models.NewBadRequest("не указан документ события")
	}
	if req.EntityID == uuid.Nil {
		return nil, models.NewBadRequest("не указана сущность события")
	}
	event, err := s.repo.Create(req)
	return dto.MapUserEvent(event), err
}

// GetCurrentUserEvents возвращает события текущего пользователя.
func (s *UserEventService) GetCurrentUserEvents(filter models.UserEventFilter) (*dto.PagedResult[dto.UserEvent], error) {
	userID, err := s.currentUserUUID()
	if err != nil {
		return nil, err
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	result, err := s.repo.GetList(userID, filter)
	if err != nil {
		return nil, err
	}
	return &dto.PagedResult[dto.UserEvent]{
		Items:      dto.MapUserEvents(result.Items),
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PageSize:   result.PageSize,
	}, nil
}

// GetUnreadCount возвращает количество непрочитанных событий текущего пользователя.
func (s *UserEventService) GetUnreadCount() (int, error) {
	userID, err := s.currentUserUUID()
	if err != nil {
		return 0, err
	}
	return s.repo.CountUnread(userID)
}

// MarkRead отмечает событие текущего пользователя прочитанным.
func (s *UserEventService) MarkRead(id string) error {
	eventID, err := uuid.Parse(id)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID события", err)
	}
	userID, err := s.currentUserUUID()
	if err != nil {
		return err
	}
	return s.repo.MarkRead(eventID, userID, time.Now())
}

// MarkDocumentRead отмечает все события текущего пользователя по документу прочитанными.
func (s *UserEventService) MarkDocumentRead(documentID string) error {
	docID, err := uuid.Parse(documentID)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID документа", err)
	}
	userID, err := s.currentUserUUID()
	if err != nil {
		return err
	}
	return s.repo.MarkDocumentRead(docID, userID, time.Now())
}

// MarkAllRead отмечает все события текущего пользователя прочитанными.
func (s *UserEventService) MarkAllRead() error {
	userID, err := s.currentUserUUID()
	if err != nil {
		return err
	}
	return s.repo.MarkAllRead(userID, time.Now())
}

func (s *UserEventService) currentUserUUID() (uuid.UUID, error) {
	if s.auth == nil {
		return uuid.Nil, ErrNotAuthenticated
	}
	return s.auth.GetCurrentUserUUID()
}

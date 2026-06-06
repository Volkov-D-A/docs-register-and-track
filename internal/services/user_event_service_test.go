package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

type fakeUserEventStore struct {
	created      []models.CreateUserEventRequest
	items        []models.UserEvent
	unreadCount  int
	markedRead   []uuid.UUID
	markedDocFor []uuid.UUID
	markedAllFor []uuid.UUID
}

func (s *fakeUserEventStore) Create(req models.CreateUserEventRequest) (*models.UserEvent, error) {
	s.created = append(s.created, req)
	event := &models.UserEvent{
		ID:              uuid.New(),
		RecipientUserID: req.RecipientUserID,
		ActorUserID:     req.ActorUserID,
		DocumentID:      req.DocumentID,
		DocumentKind:    req.DocumentKind,
		DocumentNumber:  req.DocumentNumber,
		EntityType:      req.EntityType,
		EntityID:        req.EntityID,
		EventType:       req.EventType,
		Title:           req.Title,
		Message:         req.Message,
		Metadata:        req.Metadata,
		CreatedAt:       time.Now(),
	}
	s.items = append([]models.UserEvent{*event}, s.items...)
	return event, nil
}

func (s *fakeUserEventStore) GetByID(id uuid.UUID) (*models.UserEvent, error) {
	for i := range s.items {
		if s.items[i].ID == id {
			return &s.items[i], nil
		}
	}
	return nil, nil
}

func (s *fakeUserEventStore) GetList(userID uuid.UUID, filter models.UserEventFilter) (*models.PagedResult[models.UserEvent], error) {
	items := make([]models.UserEvent, 0)
	for _, item := range s.items {
		if item.RecipientUserID != userID {
			continue
		}
		if filter.UnreadOnly && item.ReadAt != nil {
			continue
		}
		items = append(items, item)
	}
	return &models.PagedResult[models.UserEvent]{
		Items:      items,
		TotalCount: len(items),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

func (s *fakeUserEventStore) CountUnread(userID uuid.UUID) (int, error) {
	if s.unreadCount > 0 {
		return s.unreadCount, nil
	}
	count := 0
	for _, item := range s.items {
		if item.RecipientUserID == userID && item.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (s *fakeUserEventStore) MarkRead(id, userID uuid.UUID, readAt time.Time) error {
	s.markedRead = append(s.markedRead, id)
	for i := range s.items {
		if s.items[i].ID == id && s.items[i].RecipientUserID == userID {
			s.items[i].ReadAt = &readAt
			return nil
		}
	}
	return models.NewNotFound("событие не найдено")
}

func (s *fakeUserEventStore) MarkAllRead(userID uuid.UUID, readAt time.Time) error {
	s.markedAllFor = append(s.markedAllFor, userID)
	for i := range s.items {
		if s.items[i].RecipientUserID == userID && s.items[i].ReadAt == nil {
			s.items[i].ReadAt = &readAt
		}
	}
	return nil
}

func (s *fakeUserEventStore) MarkDocumentRead(documentID, userID uuid.UUID, readAt time.Time) error {
	s.markedDocFor = append(s.markedDocFor, documentID)
	for i := range s.items {
		if s.items[i].DocumentID == documentID && s.items[i].RecipientUserID == userID && s.items[i].ReadAt == nil {
			s.items[i].ReadAt = &readAt
		}
	}
	return nil
}

func setupUserEventService(t *testing.T) (*UserEventService, *fakeUserEventStore, *models.User) {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	password := "Passw0rd!"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)

	user := &models.User{
		ID:           uuid.New(),
		Login:        "events_user",
		PasswordHash: hash,
		IsActive:     true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err = auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	store := &fakeUserEventStore{}
	return NewUserEventService(store, auth), store, user
}

func TestUserEventService_GetCurrentUserEvents(t *testing.T) {
	svc, store, user := setupUserEventService(t)
	otherUserID := uuid.New()
	eventID := uuid.New()
	store.items = []models.UserEvent{
		{
			ID:              eventID,
			RecipientUserID: user.ID,
			DocumentID:      uuid.New(),
			EntityID:        uuid.New(),
			DocumentKind:    "incoming_letter",
			EntityType:      models.UserEventEntityAssignment,
			EventType:       models.UserEventAssignmentCreated,
			Title:           "Новое поручение",
			Message:         "Текст события",
			CreatedAt:       time.Now(),
		},
		{
			ID:              uuid.New(),
			RecipientUserID: otherUserID,
			DocumentID:      uuid.New(),
			EntityID:        uuid.New(),
			DocumentKind:    "incoming_letter",
			EntityType:      models.UserEventEntityAssignment,
			EventType:       models.UserEventAssignmentCreated,
			Title:           "Чужое событие",
			Message:         "Не должно попасть в ответ",
			CreatedAt:       time.Now(),
		},
	}

	result, err := svc.GetCurrentUserEvents(models.UserEventFilter{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PageSize)
	require.Len(t, result.Items, 1)
	assert.Equal(t, eventID.String(), result.Items[0].ID)
}

func TestUserEventService_ReadMarkers(t *testing.T) {
	svc, store, user := setupUserEventService(t)
	eventID := uuid.New()
	documentID := uuid.New()
	store.items = []models.UserEvent{
		{
			ID:              eventID,
			RecipientUserID: user.ID,
			DocumentID:      documentID,
			EntityID:        uuid.New(),
			DocumentKind:    "incoming_letter",
			EntityType:      models.UserEventEntityAssignment,
			EventType:       models.UserEventAssignmentCreated,
			CreatedAt:       time.Now(),
		},
	}

	count, err := svc.GetUnreadCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	err = svc.MarkRead(eventID.String())
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{eventID}, store.markedRead)

	err = svc.MarkDocumentRead(documentID.String())
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{documentID}, store.markedDocFor)

	err = svc.MarkAllRead()
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{user.ID}, store.markedAllFor)
}

func TestUserEventService_CreateValidation(t *testing.T) {
	svc, _, _ := setupUserEventService(t)

	event, err := svc.create(models.CreateUserEventRequest{
		DocumentID: uuid.New(),
		EntityID:   uuid.New(),
	})
	require.Error(t, err)
	requireAppError(t, err, "VALIDATION_ERROR", 400, "не указан получатель события")
	assert.Nil(t, event)

	event, err = svc.create(models.CreateUserEventRequest{
		RecipientUserID: uuid.New(),
		EntityID:        uuid.New(),
	})
	require.Error(t, err)
	requireAppError(t, err, "VALIDATION_ERROR", 400, "не указан документ события")
	assert.Nil(t, event)
}

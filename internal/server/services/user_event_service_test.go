package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/mocks"
)

type fakeUserEventStore struct {
	items        map[uuid.UUID][]models.UserEvent
	markedDocFor []uuid.UUID
	markedAllFor []uuid.UUID
}

func (s *fakeUserEventStore) GetList(userID uuid.UUID, filter models.UserEventFilter) (*models.PagedResult[models.UserEvent], error) {
	items := make([]models.UserEvent, 0)
	items = append(items, s.items[userID]...)
	return &models.PagedResult[models.UserEvent]{
		Items:      items,
		TotalCount: len(items),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

func (s *fakeUserEventStore) CountUnread(userID uuid.UUID) (int, error) {
	count := 0
	for _, item := range s.items[userID] {
		if item.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (s *fakeUserEventStore) MarkAllRead(userID uuid.UUID, readAt time.Time) error {
	s.markedAllFor = append(s.markedAllFor, userID)
	for i := range s.items[userID] {
		if s.items[userID][i].ReadAt == nil {
			s.items[userID][i].ReadAt = &readAt
		}
	}
	return nil
}

func (s *fakeUserEventStore) MarkDocumentRead(documentID, userID uuid.UUID, readAt time.Time) error {
	s.markedDocFor = append(s.markedDocFor, documentID)
	for i := range s.items[userID] {
		if s.items[userID][i].DocumentID == documentID && s.items[userID][i].ReadAt == nil {
			s.items[userID][i].ReadAt = &readAt
		}
	}
	return nil
}

func setupUserEventService(t *testing.T) (*UserEventService, *fakeUserEventStore, *models.User) {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := newDocumentPrincipalStub(userRepo)

	user := &models.User{
		ID:    uuid.New(),
		Login: "events_user",

		IsActive: true,
	}
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	store := &fakeUserEventStore{}
	return NewUserEventService(store, auth), store, user
}

func TestUserEventService_GetCurrentUserEvents(t *testing.T) {
	svc, store, user := setupUserEventService(t)
	otherUserID := uuid.New()
	eventID := uuid.New()
	store.items = map[uuid.UUID][]models.UserEvent{
		user.ID: {{
			ID:           eventID,
			DocumentID:   uuid.New(),
			DocumentKind: "incoming_letter",
			EntityType:   "deadline_control",
			EventType:    "deadline_control_create",
			Title:        "Новое поручение",
			Message:      "Текст события",
			CreatedAt:    time.Now(),
		}},
		otherUserID: {{
			ID:           uuid.New(),
			DocumentID:   uuid.New(),
			DocumentKind: "incoming_letter",
			EntityType:   "deadline_control",
			EventType:    "deadline_control_create",
			Title:        "Чужое событие",
			Message:      "Не должно попасть в ответ",
			CreatedAt:    time.Now(),
		}},
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
	store.items = map[uuid.UUID][]models.UserEvent{
		user.ID: {{
			ID:           eventID,
			DocumentID:   documentID,
			DocumentKind: "incoming_letter",
			EntityType:   "deadline_control",
			EventType:    "deadline_control_create",
			CreatedAt:    time.Now(),
		}},
	}

	count, err := svc.GetUnreadCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	err = svc.MarkDocumentRead(documentID.String())
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{documentID}, store.markedDocFor)
	count, err = svc.GetUnreadCount()
	require.NoError(t, err)
	assert.Zero(t, count)

	err = svc.MarkAllRead()
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{user.ID}, store.markedAllFor)
}

func TestUserEventServiceReadMarkersRejectOtherRecipient(t *testing.T) {
	svc, store, user := setupUserEventService(t)
	documentID := uuid.New()
	foreignID := uuid.New()
	foreignUserID := uuid.New()
	store.items = map[uuid.UUID][]models.UserEvent{
		user.ID:       {{ID: uuid.New(), DocumentID: documentID}},
		foreignUserID: {{ID: foreignID, DocumentID: documentID}},
	}
	require.NoError(t, svc.MarkDocumentRead(documentID.String()))
	require.NotNil(t, store.items[user.ID][0].ReadAt)
	require.Nil(t, store.items[foreignUserID][0].ReadAt)
	require.NoError(t, svc.MarkAllRead())
	require.Nil(t, store.items[foreignUserID][0].ReadAt)
}

func TestUserEventServiceRequiresPrincipal(t *testing.T) {
	svc := NewUserEventService(nil, nil)
	_, err := svc.GetCurrentUserEvents(models.UserEventFilter{})
	require.ErrorIs(t, err, models.ErrUnauthorized)
	_, err = svc.GetUnreadCount()
	require.ErrorIs(t, err, models.ErrUnauthorized)
	require.ErrorIs(t, svc.MarkDocumentRead(uuid.NewString()), models.ErrUnauthorized)
	require.ErrorIs(t, svc.MarkAllRead(), models.ErrUnauthorized)
}

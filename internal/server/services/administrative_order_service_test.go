package services

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/mocks"
)

type administrativeOrderServiceStore struct {
	person       *models.AdministrativeOrderAcknowledgmentPerson
	personErr    error
	marked       *models.AdministrativeOrderAcknowledgmentPerson
	markErr      error
	lastPersonID uuid.UUID
	lastMarkerID uuid.UUID
	effects      []models.OutboxEvent
}

func (s *administrativeOrderServiceStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.AdministrativeOrderDocument], error) {
	return nil, nil
}

func (s *administrativeOrderServiceStore) GetByID(id uuid.UUID) (*models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *administrativeOrderServiceStore) GetByIDs(ids []uuid.UUID) ([]models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *administrativeOrderServiceStore) GetAcknowledgmentPersonByID(id uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	s.lastPersonID = id
	if s.personErr != nil {
		return nil, s.personErr
	}
	return s.person, nil
}

func (s *administrativeOrderServiceStore) GetAcknowledgmentPeople(documentID uuid.UUID) ([]models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *administrativeOrderServiceStore) MarkAcknowledgmentPersonWithOutbox(id uuid.UUID, acknowledgedBy uuid.UUID, effects []models.OutboxEvent) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	s.lastPersonID = id
	s.lastMarkerID = acknowledgedBy
	s.effects = effects
	if s.markErr != nil {
		return nil, s.markErr
	}
	return s.marked, nil
}

func setupAdministrativeOrderService(t *testing.T, allowed map[models.DocumentKind]map[string]bool) (*AdministrativeOrderService, *administrativeOrderServiceStore, *documentAccessDocumentStore, *models.User) {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := newDocumentPrincipalStub(userRepo)
	user := documentAccessUser(false, nil)
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	store := &administrativeOrderServiceStore{}
	docRepo := &documentAccessDocumentStore{docs: map[uuid.UUID]models.Document{}}
	access := NewDocumentAccessService(auth, &kindActionDocumentAccessStore{allowed: allowed, visible: func(id uuid.UUID) bool {
		doc := docRepo.docs[id]
		return allowed[doc.Kind]["register"] || allowed[doc.Kind]["control_deadlines"]
	}}, docRepo)
	return NewAdministrativeOrderService(store, auth, access), store, docRepo, user
}

func TestAdministrativeOrderService_MarkAcknowledged(t *testing.T) {
	t.Run("marks person and writes journal entry", func(t *testing.T) {
		documentID := uuid.New()
		personID := uuid.New()
		now := time.Now()
		svc, store, docRepo, user := setupAdministrativeOrderService(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "control_deadlines", "register"),
		)
		docRepo.docs[documentID] = documentAccessDoc(documentID, models.DocumentKindAdministrativeOrder)
		store.person = &models.AdministrativeOrderAcknowledgmentPerson{
			ID:         personID,
			DocumentID: documentID,
			FullName:   "Иван Иванов",
		}
		store.marked = &models.AdministrativeOrderAcknowledgmentPerson{
			ID:                 personID,
			DocumentID:         documentID,
			FullName:           "Иван Иванов",
			AcknowledgedAt:     &now,
			AcknowledgedBy:     &user.ID,
			AcknowledgedByName: "Текущий пользователь",
		}
		result, err := svc.MarkAcknowledged(personID.String())

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, personID.String(), result.ID)
		assert.Equal(t, "Иван Иванов", result.FullName)
		assert.Equal(t, personID, store.lastPersonID)
		assert.Equal(t, user.ID, store.lastMarkerID)
	})

	t.Run("rejects invalid id", func(t *testing.T) {
		svc, _, _, _ := setupAdministrativeOrderService(t, nil)

		result, err := svc.MarkAcknowledged("bad-id")

		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID строки ознакомления")
		assert.Nil(t, result)
	})

	t.Run("returns not found for missing person", func(t *testing.T) {
		personID := uuid.New()
		svc, store, _, _ := setupAdministrativeOrderService(t, nil)
		store.person = nil

		result, err := svc.MarkAcknowledged(personID.String())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "строка ознакомления не найдена")
	})

	t.Run("requires update access to order", func(t *testing.T) {
		personID := uuid.New()
		documentID := uuid.New()
		svc, store, docRepo, _ := setupAdministrativeOrderService(t, nil)
		docRepo.docs[documentID] = documentAccessDoc(documentID, models.DocumentKindAdministrativeOrder)
		store.person = &models.AdministrativeOrderAcknowledgmentPerson{
			ID:         personID,
			DocumentID: documentID,
			FullName:   "Иван Иванов",
		}

		result, err := svc.MarkAcknowledged(personID.String())

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
		assert.Equal(t, uuid.Nil, store.lastMarkerID)
	})

	t.Run("returns repository mark error", func(t *testing.T) {
		personID := uuid.New()
		repoErr := errors.New("mark failed")
		documentID := uuid.New()
		svc, store, docRepo, _ := setupAdministrativeOrderService(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "control_deadlines", "register"),
		)
		docRepo.docs[documentID] = documentAccessDoc(documentID, models.DocumentKindAdministrativeOrder)
		store.person = &models.AdministrativeOrderAcknowledgmentPerson{
			ID:         personID,
			DocumentID: documentID,
			FullName:   "Иван Иванов",
		}
		store.markErr = repoErr

		result, err := svc.MarkAcknowledged(personID.String())

		require.ErrorIs(t, err, repoErr)
		assert.Nil(t, result)
	})
}

func TestAdministrativeOrderService_MarkAcknowledged_EnqueuesJournalWithPrincipal(t *testing.T) {
	svc, store, docRepo, user := setupAdministrativeOrderService(
		t,
		allowDocumentActions(models.DocumentKindAdministrativeOrder, "control_deadlines", "register"),
	)
	personID := uuid.New()
	documentID := uuid.New()
	docRepo.docs[documentID] = documentAccessDoc(documentID, models.DocumentKindAdministrativeOrder)
	store.person = &models.AdministrativeOrderAcknowledgmentPerson{ID: personID, DocumentID: documentID, FullName: "Петр Петров"}
	store.marked = store.person
	result, err := svc.MarkAcknowledged(personID.String())

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, store.effects, 1)
	assert.Equal(t, models.OutboxEventJournal, store.effects[0].EventType)
	var journal models.CreateJournalEntryRequest
	require.NoError(t, json.Unmarshal([]byte(store.effects[0].Payload), &journal))
	assert.Equal(t, documentID, journal.DocumentID)
	assert.Equal(t, user.ID, journal.UserID)
	assert.Equal(t, "ORDER_ACKNOWLEDGE", journal.Action)
	assert.Contains(t, journal.Details, "Петр Петров")
}

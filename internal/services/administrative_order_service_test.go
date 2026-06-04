package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type administrativeOrderServiceStore struct {
	person       *models.AdministrativeOrderAcknowledgmentPerson
	personErr    error
	marked       *models.AdministrativeOrderAcknowledgmentPerson
	markErr      error
	lastPersonID uuid.UUID
	lastMarkerID uuid.UUID
}

func (s *administrativeOrderServiceStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.AdministrativeOrderDocument], error) {
	return nil, nil
}

func (s *administrativeOrderServiceStore) GetByID(id uuid.UUID) (*models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *administrativeOrderServiceStore) Create(req models.CreateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *administrativeOrderServiceStore) Update(req models.UpdateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
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

func (s *administrativeOrderServiceStore) MarkAcknowledgmentPerson(id uuid.UUID, acknowledgedBy uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	s.lastPersonID = id
	s.lastMarkerID = acknowledgedBy
	if s.markErr != nil {
		return nil, s.markErr
	}
	return s.marked, nil
}

func (s *administrativeOrderServiceStore) CancelByLink(id uuid.UUID, cancelledAt time.Time) error {
	return nil
}

func (s *administrativeOrderServiceStore) GetCount() (int, error) {
	return 0, nil
}

func setupAdministrativeOrderService(t *testing.T, allowed map[models.DocumentKind]map[string]bool) (*AdministrativeOrderService, *administrativeOrderServiceStore, *documentAccessDocumentStore, *mocks.JournalStore, *models.User) {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	user := documentAccessUser(false, nil)
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	store := &administrativeOrderServiceStore{}
	docRepo := &documentAccessDocumentStore{docs: map[uuid.UUID]models.Document{}}
	access := NewDocumentAccessService(
		auth,
		&documentAccessDepartmentStore{},
		&documentAccessAssignmentStore{accessible: map[uuid.UUID]struct{}{}},
		&documentAccessAcknowledgmentStore{accessible: map[uuid.UUID]struct{}{}},
		&kindActionDocumentAccessStore{allowed: allowed},
		docRepo,
		nil,
		nil,
	)
	journalRepo := mocks.NewJournalStore(t)
	journal := NewJournalService(journalRepo, auth, access)

	return NewAdministrativeOrderService(store, auth, access, journal), store, docRepo, journalRepo, user
}

func TestAdministrativeOrderService_MarkAcknowledged(t *testing.T) {
	t.Run("marks person and writes journal entry", func(t *testing.T) {
		documentID := uuid.New()
		personID := uuid.New()
		now := time.Now()
		svc, store, docRepo, journalRepo, user := setupAdministrativeOrderService(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "read", "update"),
		)
		docRepo.docs[documentID] = documentAccessDoc(documentID, uuid.New(), models.DocumentKindAdministrativeOrder)
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
		journalRepo.On("Create", mock.Anything, mock.MatchedBy(func(req models.CreateJournalEntryRequest) bool {
			return req.DocumentID == documentID &&
				req.UserID == user.ID &&
				req.Action == "ORDER_ACKNOWLEDGE" &&
				req.Details == "Ознакомлен: Иван Иванов"
		})).Return(uuid.New(), nil).Once()

		result, err := svc.MarkAcknowledged(personID.String())

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, personID.String(), result.ID)
		assert.Equal(t, "Иван Иванов", result.FullName)
		assert.Equal(t, personID, store.lastPersonID)
		assert.Equal(t, user.ID, store.lastMarkerID)
	})

	t.Run("rejects invalid id", func(t *testing.T) {
		svc, _, _, _, _ := setupAdministrativeOrderService(t, nil)

		result, err := svc.MarkAcknowledged("bad-id")

		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID строки ознакомления")
		assert.Nil(t, result)
	})

	t.Run("returns not found for missing person", func(t *testing.T) {
		personID := uuid.New()
		svc, store, _, _, _ := setupAdministrativeOrderService(t, nil)
		store.person = nil

		result, err := svc.MarkAcknowledged(personID.String())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "строка ознакомления не найдена")
	})

	t.Run("requires update access to order", func(t *testing.T) {
		personID := uuid.New()
		documentID := uuid.New()
		svc, store, docRepo, _, _ := setupAdministrativeOrderService(t, nil)
		docRepo.docs[documentID] = documentAccessDoc(documentID, uuid.New(), models.DocumentKindAdministrativeOrder)
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
		svc, store, docRepo, _, _ := setupAdministrativeOrderService(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "read", "update"),
		)
		docRepo.docs[documentID] = documentAccessDoc(documentID, uuid.New(), models.DocumentKindAdministrativeOrder)
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

func TestAdministrativeOrderService_MarkAcknowledged_JournalContext(t *testing.T) {
	svc, store, docRepo, journalRepo, _ := setupAdministrativeOrderService(
		t,
		allowDocumentActions(models.DocumentKindAdministrativeOrder, "read", "update"),
	)
	personID := uuid.New()
	documentID := uuid.New()
	docRepo.docs[documentID] = documentAccessDoc(documentID, uuid.New(), models.DocumentKindAdministrativeOrder)
	store.person = &models.AdministrativeOrderAcknowledgmentPerson{ID: personID, DocumentID: documentID, FullName: "Петр Петров"}
	store.marked = store.person
	journalRepo.On("Create", mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }), mock.Anything).
		Return(uuid.New(), nil).Once()

	result, err := svc.MarkAcknowledged(personID.String())

	require.NoError(t, err)
	require.NotNil(t, result)
}

package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type administrativeOrderHandlerDeps struct {
	handler     *AdministrativeOrderCommandHandler
	repo        *administrativeOrderCommandStore
	journalRepo *mocks.JournalStore
	auth        *AuthService
	user        *models.User
}

type administrativeOrderCommandStore struct {
	createReq    *models.CreateAdministrativeOrderDocRequest
	createResult *models.AdministrativeOrderDocument
	createErr    error
	updateReq    *models.UpdateAdministrativeOrderDocRequest
	updateResult *models.AdministrativeOrderDocument
	updateErr    error
}

func (s *administrativeOrderCommandStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.AdministrativeOrderDocument], error) {
	return nil, nil
}

func (s *administrativeOrderCommandStore) GetByID(id uuid.UUID) (*models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *administrativeOrderCommandStore) Create(req models.CreateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	s.createReq = &req
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createResult != nil {
		return s.createResult, nil
	}
	return &models.AdministrativeOrderDocument{ID: uuid.New()}, nil
}

func (s *administrativeOrderCommandStore) Update(req models.UpdateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	s.updateReq = &req
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	if s.updateResult != nil {
		return s.updateResult, nil
	}
	return &models.AdministrativeOrderDocument{ID: req.ID}, nil
}

func (s *administrativeOrderCommandStore) GetAcknowledgmentPersonByID(id uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *administrativeOrderCommandStore) GetAcknowledgmentPeople(documentID uuid.UUID) ([]models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *administrativeOrderCommandStore) MarkAcknowledgmentPerson(id uuid.UUID, acknowledgedBy uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *administrativeOrderCommandStore) CancelByLink(id uuid.UUID, cancelledAt time.Time) error {
	return nil
}

func (s *administrativeOrderCommandStore) GetCount() (int, error) {
	return 0, nil
}

func setupAdministrativeOrderCommandHandler(t *testing.T, allowed map[models.DocumentKind]map[string]bool) *administrativeOrderHandlerDeps {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	user := documentAccessUser(false, nil)
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	repo := &administrativeOrderCommandStore{}
	nomRepo := mocks.NewNomenclatureStore(t)
	journalRepo := mocks.NewJournalStore(t)
	access := NewDocumentAccessService(
		auth,
		&documentAccessDepartmentStore{},
		&documentAccessAssignmentStore{accessible: map[uuid.UUID]struct{}{}},
		&documentAccessAcknowledgmentStore{accessible: map[uuid.UUID]struct{}{}},
		&kindActionDocumentAccessStore{allowed: allowed},
		nil,
		nil,
		nil,
	)
	journal := NewJournalService(journalRepo, auth, access)
	handler := NewAdministrativeOrderCommandHandler(repo, nomRepo, auth, journal, access)

	return &administrativeOrderHandlerDeps{
		handler:     handler,
		repo:        repo,
		journalRepo: journalRepo,
		auth:        auth,
		user:        user,
	}
}

func validAdministrativeOrderRegisterRequest(nomenclatureID, idempotencyKey uuid.UUID) AdministrativeOrderRegisterRequest {
	return AdministrativeOrderRegisterRequest{
		NomenclatureID:          nomenclatureID.String(),
		IdempotencyKey:          idempotencyKey.String(),
		OrderDate:               "2026-06-03",
		Title:                   " О назначении ответственного ",
		ExecutionController:     " Контрольный отдел ",
		ExecutionDeadline:       "2026-06-30",
		IsActive:                true,
		AcknowledgmentFullNames: []string{" Иван Иванов ", "", " Петр Петров "},
		RegistrationNumber:      " ORD-10 ",
	}
}

func TestAdministrativeOrderCommandHandler_Register(t *testing.T) {
	t.Run("creates active order and writes journal entry", func(t *testing.T) {
		nomenclatureID := uuid.New()
		idempotencyKey := uuid.New()
		documentID := uuid.New()
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "create"),
		)
		req := validAdministrativeOrderRegisterRequest(nomenclatureID, idempotencyKey)
		deps.repo.createResult = &models.AdministrativeOrderDocument{
			ID:                  documentID,
			NomenclatureID:      nomenclatureID,
			OrderNumber:         "ORD-10",
			OrderDate:           time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
			Title:               "О назначении ответственного",
			ExecutionController: "Контрольный отдел",
			IsActive:            true,
			CreatedBy:           deps.user.ID,
		}

		deps.journalRepo.On("Create", context.Background(), mock.MatchedBy(func(journalReq models.CreateJournalEntryRequest) bool {
			return journalReq.DocumentID == documentID &&
				journalReq.UserID == deps.user.ID &&
				journalReq.Action == "CREATE" &&
				journalReq.Details == "Приказ зарегистрирован. Рег. номер: ORD-10"
		})).Return(uuid.New(), nil).Once()

		result, err := deps.handler.Register(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, deps.repo.createReq)
		assert.Equal(t, documentID.String(), result.ID)
		assert.Equal(t, nomenclatureID, deps.repo.createReq.NomenclatureID)
		assert.Equal(t, idempotencyKey, deps.repo.createReq.IdempotencyKey)
		assert.Equal(t, deps.user.ID, deps.repo.createReq.CreatedBy)
		assert.Equal(t, "ORD-10", deps.repo.createReq.OrderNumber)
		assert.Equal(t, "О назначении ответственного", deps.repo.createReq.Title)
		assert.Equal(t, "Контрольный отдел", deps.repo.createReq.ExecutionController)
		require.NotNil(t, deps.repo.createReq.ExecutionDeadline)
		assert.Equal(t, time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), *deps.repo.createReq.ExecutionDeadline)
		assert.True(t, deps.repo.createReq.IsActive)
		assert.Nil(t, deps.repo.createReq.CancelledAt)
		assert.Equal(t, []string{"Иван Иванов", "Петр Петров"}, deps.repo.createReq.AcknowledgmentFullNames)
	})

	t.Run("rejects missing create permission", func(t *testing.T) {
		deps := setupAdministrativeOrderCommandHandler(t, nil)
		req := validAdministrativeOrderRegisterRequest(uuid.New(), uuid.New())

		result, err := deps.handler.Register(req)

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})

	t.Run("rejects active order with cancellation date", func(t *testing.T) {
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "create"),
		)
		req := validAdministrativeOrderRegisterRequest(uuid.New(), uuid.New())
		req.CancelledAt = "2026-07-01"

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "для действующего приказа дата отмены должна быть пустой")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})

	t.Run("rejects inactive order without cancellation date", func(t *testing.T) {
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "create"),
		)
		req := validAdministrativeOrderRegisterRequest(uuid.New(), uuid.New())
		req.IsActive = false

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "для недействующего приказа укажите дату отмены")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})

	t.Run("rejects empty execution controller", func(t *testing.T) {
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "create"),
		)
		req := validAdministrativeOrderRegisterRequest(uuid.New(), uuid.New())
		req.ExecutionController = "  "

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "укажите контроль за выполнением")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})

	t.Run("rejects invalid deadline date", func(t *testing.T) {
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "create"),
		)
		req := validAdministrativeOrderRegisterRequest(uuid.New(), uuid.New())
		req.ExecutionDeadline = "30.06.2026"

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный формат срока выполнения")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})
}

func TestAdministrativeOrderCommandHandler_Update(t *testing.T) {
	t.Run("updates inactive order and writes journal entry", func(t *testing.T) {
		documentID := uuid.New()
		cancelledAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindAdministrativeOrder),
			},
		}
		deps.repo.updateResult = &models.AdministrativeOrderDocument{
			ID:                  documentID,
			OrderNumber:         "ORD-10",
			OrderDate:           time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
			Title:               "Обновленный приказ",
			ExecutionController: "Контроль",
			IsActive:            false,
			CancelledAt:         &cancelledAt,
			CreatedBy:           deps.user.ID,
		}
		req := AdministrativeOrderUpdateRequest{
			ID:                      documentID.String(),
			OrderDate:               "2026-06-04",
			Title:                   " Обновленный приказ ",
			ExecutionController:     " Контроль ",
			IsActive:                false,
			CancelledAt:             "2026-07-01T00:00:00Z",
			AcknowledgmentFullNames: []string{" Сидор Сидоров "},
		}

		deps.journalRepo.On("Create", context.Background(), mock.MatchedBy(func(journalReq models.CreateJournalEntryRequest) bool {
			return journalReq.DocumentID == documentID &&
				journalReq.UserID == deps.user.ID &&
				journalReq.Action == "UPDATE" &&
				journalReq.Details == "Приказ отредактирован"
		})).Return(uuid.New(), nil).Once()

		result, err := deps.handler.Update(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, deps.repo.updateReq)
		assert.Equal(t, documentID.String(), result.ID)
		assert.Equal(t, documentID, deps.repo.updateReq.ID)
		assert.Equal(t, "Обновленный приказ", deps.repo.updateReq.Title)
		assert.Equal(t, "Контроль", deps.repo.updateReq.ExecutionController)
		assert.False(t, deps.repo.updateReq.IsActive)
		require.NotNil(t, deps.repo.updateReq.CancelledAt)
		assert.Equal(t, cancelledAt, *deps.repo.updateReq.CancelledAt)
		assert.Equal(t, []string{"Сидор Сидоров"}, deps.repo.updateReq.AcknowledgmentFullNames)
	})

	t.Run("rejects invalid document ID", func(t *testing.T) {
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "read", "update"),
		)

		result, err := deps.handler.Update(AdministrativeOrderUpdateRequest{ID: "bad-id"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный ID документа")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.updateReq)
	})

	t.Run("rejects missing update permission", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "read"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindAdministrativeOrder),
			},
		}

		result, err := deps.handler.Update(AdministrativeOrderUpdateRequest{ID: documentID.String()})

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.updateReq)
	})

	t.Run("rejects invalid cancellation date", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupAdministrativeOrderCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindAdministrativeOrder, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindAdministrativeOrder),
			},
		}

		result, err := deps.handler.Update(AdministrativeOrderUpdateRequest{
			ID:                  documentID.String(),
			OrderDate:           "2026-06-04",
			ExecutionController: "Контроль",
			IsActive:            false,
			CancelledAt:         "01.07.2026",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный формат даты отмены")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.updateReq)
	})
}

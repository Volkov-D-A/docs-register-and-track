package services

import (
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

type outgoingLetterHandlerDeps struct {
	handler     *OutgoingLetterCommandHandler
	repo        *mocks.OutgoingDocStore
	refRepo     *mocks.ReferenceStore
	journalRepo *mocks.JournalStore
	auth        *AuthService
	user        *models.User
}

func setupOutgoingLetterCommandHandler(t *testing.T, allowed map[models.DocumentKind]map[string]bool) *outgoingLetterHandlerDeps {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	user := documentAccessUser(false, nil)
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	repo := mocks.NewOutgoingDocStore(t)
	refRepo := mocks.NewReferenceStore(t)
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
	handler := NewOutgoingLetterCommandHandler(repo, refRepo, nomRepo, auth, journal, access)

	return &outgoingLetterHandlerDeps{
		handler:     handler,
		repo:        repo,
		refRepo:     refRepo,
		journalRepo: journalRepo,
		auth:        auth,
		user:        user,
	}
}

func validOutgoingLetterRegisterRequest(nomenclatureID, idempotencyKey uuid.UUID) OutgoingLetterRegisterRequest {
	return OutgoingLetterRegisterRequest{
		NomenclatureID:     nomenclatureID.String(),
		IdempotencyKey:     idempotencyKey.String(),
		DocumentTypeID:     models.DocumentTypeLetter,
		RecipientOrgName:   "ООО Получатель",
		Addressee:          "Директору",
		OutgoingDate:       "2026-06-03",
		Content:            "Outgoing letter content",
		PagesCount:         2,
		SenderSignatory:    "Подписант",
		SenderExecutor:     "Исполнитель",
		RegistrationNumber: "OUT-12",
	}
}

func TestOutgoingLetterCommandHandler_Register(t *testing.T) {
	t.Run("creates outgoing letter and writes journal entry", func(t *testing.T) {
		nomenclatureID := uuid.New()
		idempotencyKey := uuid.New()
		recipientOrgID := uuid.New()
		documentID := uuid.New()
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "create"),
		)
		req := validOutgoingLetterRegisterRequest(nomenclatureID, idempotencyKey)

		deps.refRepo.On("FindOrCreateOrganization", "ООО Получатель").Return(&models.Organization{ID: recipientOrgID, Name: "ООО Получатель"}, nil).Once()
		deps.repo.On("Create", mock.MatchedBy(func(createReq models.CreateOutgoingDocRequest) bool {
			require.Equal(t, nomenclatureID, createReq.NomenclatureID)
			require.Equal(t, idempotencyKey, createReq.IdempotencyKey)
			require.Equal(t, models.DocumentTypeLetter, createReq.DocumentTypeID)
			require.Equal(t, recipientOrgID, createReq.RecipientOrgID)
			require.Equal(t, deps.user.ID, createReq.CreatedBy)
			require.Equal(t, "OUT-12", createReq.OutgoingNumber)
			require.Equal(t, "Outgoing letter content", createReq.Content)
			require.Equal(t, 2, createReq.PagesCount)
			require.Equal(t, "Подписант", createReq.SenderSignatory)
			require.Equal(t, "Исполнитель", createReq.SenderExecutor)
			require.Equal(t, "Директору", createReq.Addressee)
			return createReq.OutgoingDate.Equal(time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC))
		})).Return(&models.OutgoingDocument{
			ID:               documentID,
			NomenclatureID:   nomenclatureID,
			OutgoingNumber:   "OUT-13",
			OutgoingDate:     time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
			DocumentTypeID:   models.DocumentTypeLetter,
			RecipientOrgID:   recipientOrgID,
			RecipientOrgName: "ООО Получатель",
			Content:          req.Content,
			PagesCount:       req.PagesCount,
			CreatedBy:        deps.user.ID,
		}, nil).Once()
		result, err := deps.handler.Register(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, documentID.String(), result.ID)
		assert.Equal(t, "OUT-13", result.OutgoingNumber)
		assert.Equal(t, "ООО Получатель", result.RecipientOrgName)
	})

	t.Run("rejects missing create permission", func(t *testing.T) {
		deps := setupOutgoingLetterCommandHandler(t, nil)
		req := validOutgoingLetterRegisterRequest(uuid.New(), uuid.New())

		result, err := deps.handler.Register(req)

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})

	t.Run("rejects invalid idempotency key", func(t *testing.T) {
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "create"),
		)
		req := validOutgoingLetterRegisterRequest(uuid.New(), uuid.New())
		req.IdempotencyKey = uuid.Nil.String()

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный ключ идемпотентности")
		assert.Nil(t, result)
	})

	t.Run("rejects invalid nomenclature ID", func(t *testing.T) {
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "create"),
		)
		req := validOutgoingLetterRegisterRequest(uuid.New(), uuid.New())
		req.NomenclatureID = "bad-id"

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный ID номенклатуры")
		assert.Nil(t, result)
	})

	t.Run("rejects invalid document type", func(t *testing.T) {
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "create"),
		)
		req := validOutgoingLetterRegisterRequest(uuid.New(), uuid.New())
		req.DocumentTypeID = "unknown"

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный тип документа")
		assert.Nil(t, result)
	})

	t.Run("rejects invalid outgoing date", func(t *testing.T) {
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "create"),
		)
		req := validOutgoingLetterRegisterRequest(uuid.New(), uuid.New())
		req.OutgoingDate = "03.06.2026"
		deps.refRepo.On("FindOrCreateOrganization", "ООО Получатель").Return(&models.Organization{ID: uuid.New(), Name: "ООО Получатель"}, nil).Once()

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный формат даты исходящего документа")
		assert.Nil(t, result)
	})

	t.Run("propagates recipient organization errors", func(t *testing.T) {
		expectedErr := errors.New("reference unavailable")
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "create"),
		)
		req := validOutgoingLetterRegisterRequest(uuid.New(), uuid.New())
		deps.refRepo.On("FindOrCreateOrganization", "ООО Получатель").Return((*models.Organization)(nil), expectedErr).Once()

		result, err := deps.handler.Register(req)

		require.ErrorIs(t, err, expectedErr)
		assert.Contains(t, err.Error(), "ошибка организации получателя")
		assert.Nil(t, result)
	})

	t.Run("propagates repository error and skips journal", func(t *testing.T) {
		expectedErr := errors.New("create failed")
		recipientOrgID := uuid.New()
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "create"),
		)
		req := validOutgoingLetterRegisterRequest(uuid.New(), uuid.New())
		deps.refRepo.On("FindOrCreateOrganization", "ООО Получатель").Return(&models.Organization{ID: recipientOrgID, Name: "ООО Получатель"}, nil).Once()
		deps.repo.On("Create", mock.Anything).Return(nil, expectedErr).Once()

		result, err := deps.handler.Register(req)

		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
		deps.journalRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestOutgoingLetterCommandHandler_Update(t *testing.T) {
	t.Run("updates outgoing letter and writes journal entry", func(t *testing.T) {
		documentID := uuid.New()
		recipientOrgID := uuid.New()
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindOutgoingLetter),
			},
		}
		req := OutgoingLetterUpdateRequest{
			ID:               documentID.String(),
			DocumentTypeID:   models.DocumentTypeLetter,
			RecipientOrgName: "АО Новый получатель",
			Addressee:        "Главному инженеру",
			OutgoingDate:     "2026-06-04",
			Content:          "Updated outgoing content",
			PagesCount:       4,
			SenderSignatory:  "Новый подписант",
			SenderExecutor:   "Новый исполнитель",
		}

		deps.refRepo.On("FindOrCreateOrganization", "АО Новый получатель").Return(&models.Organization{ID: recipientOrgID, Name: "АО Новый получатель"}, nil).Once()
		deps.repo.On("Update", mock.MatchedBy(func(updateReq models.UpdateOutgoingDocRequest) bool {
			require.Equal(t, documentID, updateReq.ID)
			require.Equal(t, models.DocumentTypeLetter, updateReq.DocumentTypeID)
			require.Equal(t, recipientOrgID, updateReq.RecipientOrgID)
			require.Equal(t, "Updated outgoing content", updateReq.Content)
			require.Equal(t, 4, updateReq.PagesCount)
			require.Equal(t, "Новый подписант", updateReq.SenderSignatory)
			require.Equal(t, "Новый исполнитель", updateReq.SenderExecutor)
			require.Equal(t, "Главному инженеру", updateReq.Addressee)
			return updateReq.OutgoingDate.Equal(time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC))
		})).Return(&models.OutgoingDocument{
			ID:               documentID,
			OutgoingNumber:   "OUT-12",
			OutgoingDate:     time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
			DocumentTypeID:   models.DocumentTypeLetter,
			RecipientOrgID:   recipientOrgID,
			RecipientOrgName: "АО Новый получатель",
			Content:          req.Content,
			PagesCount:       req.PagesCount,
			CreatedBy:        deps.user.ID,
		}, nil).Once()
		result, err := deps.handler.Update(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, documentID.String(), result.ID)
		assert.Equal(t, "АО Новый получатель", result.RecipientOrgName)
		assert.Equal(t, "Updated outgoing content", result.Content)
	})

	t.Run("rejects invalid document ID", func(t *testing.T) {
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "read", "update"),
		)

		result, err := deps.handler.Update(OutgoingLetterUpdateRequest{ID: "bad-id"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный ID документа")
		assert.Nil(t, result)
	})

	t.Run("rejects missing update permission", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "read"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindOutgoingLetter),
			},
		}

		result, err := deps.handler.Update(OutgoingLetterUpdateRequest{
			ID:               documentID.String(),
			DocumentTypeID:   models.DocumentTypeLetter,
			RecipientOrgName: "АО Новый получатель",
			OutgoingDate:     "2026-06-04",
		})

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})

	t.Run("rejects invalid document type", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindOutgoingLetter),
			},
		}

		result, err := deps.handler.Update(OutgoingLetterUpdateRequest{
			ID:             documentID.String(),
			DocumentTypeID: "unknown",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный тип документа")
		assert.Nil(t, result)
	})

	t.Run("propagates recipient organization errors", func(t *testing.T) {
		documentID := uuid.New()
		expectedErr := errors.New("reference unavailable")
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindOutgoingLetter),
			},
		}
		deps.refRepo.On("FindOrCreateOrganization", "АО Новый получатель").Return((*models.Organization)(nil), expectedErr).Once()

		result, err := deps.handler.Update(OutgoingLetterUpdateRequest{
			ID:               documentID.String(),
			DocumentTypeID:   models.DocumentTypeLetter,
			RecipientOrgName: "АО Новый получатель",
		})

		require.ErrorIs(t, err, expectedErr)
		assert.Contains(t, err.Error(), "ошибка организации получателя")
		assert.Nil(t, result)
	})

	t.Run("rejects invalid outgoing date", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindOutgoingLetter),
			},
		}
		deps.refRepo.On("FindOrCreateOrganization", "АО Новый получатель").Return(&models.Organization{ID: uuid.New(), Name: "АО Новый получатель"}, nil).Once()

		result, err := deps.handler.Update(OutgoingLetterUpdateRequest{
			ID:               documentID.String(),
			DocumentTypeID:   models.DocumentTypeLetter,
			RecipientOrgName: "АО Новый получатель",
			OutgoingDate:     "04.06.2026",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный формат даты исходящего документа")
		assert.Nil(t, result)
	})

	t.Run("propagates repository error and skips journal", func(t *testing.T) {
		documentID := uuid.New()
		expectedErr := errors.New("update failed")
		deps := setupOutgoingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindOutgoingLetter, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindOutgoingLetter),
			},
		}
		deps.refRepo.On("FindOrCreateOrganization", "АО Новый получатель").Return(&models.Organization{ID: uuid.New(), Name: "АО Новый получатель"}, nil).Once()
		deps.repo.On("Update", mock.Anything).Return(nil, expectedErr).Once()

		result, err := deps.handler.Update(OutgoingLetterUpdateRequest{
			ID:               documentID.String(),
			DocumentTypeID:   models.DocumentTypeLetter,
			RecipientOrgName: "АО Новый получатель",
			OutgoingDate:     "2026-06-04",
		})

		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
		deps.journalRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestOutgoingLetterCommandHandler_CommandInterface(t *testing.T) {
	deps := setupOutgoingLetterCommandHandler(
		t,
		allowDocumentActions(models.DocumentKindOutgoingLetter, "create", "read", "update"),
	)

	registered, err := deps.handler.RegisterDocument(struct{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid register request")
	assert.Nil(t, registered)

	updated, err := deps.handler.UpdateDocument(struct{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid update request")
	assert.Nil(t, updated)
}

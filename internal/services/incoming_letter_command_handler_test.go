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

type incomingLetterHandlerDeps struct {
	handler     *IncomingLetterCommandHandler
	repo        *mocks.IncomingDocStore
	refRepo     *mocks.ReferenceStore
	journalRepo *mocks.JournalStore
	auth        *AuthService
	user        *models.User
}

func setupIncomingLetterCommandHandler(t *testing.T, allowed map[models.DocumentKind]map[string]bool) *incomingLetterHandlerDeps {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	user := documentAccessUser(false, nil)
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	repo := mocks.NewIncomingDocStore(t)
	nomRepo := mocks.NewNomenclatureStore(t)
	refRepo := mocks.NewReferenceStore(t)
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
	handler := NewIncomingLetterCommandHandler(repo, nomRepo, refRepo, auth, journal, access)

	return &incomingLetterHandlerDeps{
		handler:     handler,
		repo:        repo,
		refRepo:     refRepo,
		journalRepo: journalRepo,
		auth:        auth,
		user:        user,
	}
}

func validIncomingLetterRegisterRequest(nomenclatureID, idempotencyKey uuid.UUID) IncomingLetterRegisterRequest {
	return IncomingLetterRegisterRequest{
		NomenclatureID:     nomenclatureID.String(),
		IdempotencyKey:     idempotencyKey.String(),
		DocumentTypeID:     models.DocumentTypeLetter,
		IncomingDate:       "2026-06-03",
		Content:            "Incoming letter content",
		PagesCount:         3,
		SenderSignatory:    "Sender",
		RegistrationNumber: "12/26",
		Correspondents: []IncomingLetterCorrespondentRequest{
			{
				RegistrationNumber: "A-1",
				RegistrationDate:   "2026-06-02",
				CorrespondentName:  "ООО Ромашка",
			},
		},
	}
}

func TestIncomingLetterCommandHandler_Register(t *testing.T) {
	t.Run("creates incoming letter and writes journal entry", func(t *testing.T) {
		nomenclatureID := uuid.New()
		idempotencyKey := uuid.New()
		orgID := uuid.New()
		documentID := uuid.New()
		deps := setupIncomingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindIncomingLetter, "create"),
		)
		req := validIncomingLetterRegisterRequest(nomenclatureID, idempotencyKey)
		req.Resolution = "Рассмотреть"
		req.ResolutionAuthor = "Руководитель"
		req.ResolutionExecutors = "Иванов; Петров"

		deps.refRepo.On("FindOrCreateResolutionExecutor", "Иванов").Return(&models.ResolutionExecutor{ID: uuid.New(), Name: "Иванов"}, nil).Once()
		deps.refRepo.On("FindOrCreateResolutionExecutor", "Петров").Return(&models.ResolutionExecutor{ID: uuid.New(), Name: "Петров"}, nil).Once()
		deps.refRepo.On("FindOrCreateOrganization", "ООО Ромашка").Return(&models.Organization{ID: orgID, Name: "ООО Ромашка"}, nil).Once()
		deps.repo.On("Create", mock.MatchedBy(func(createReq models.CreateIncomingDocRequest) bool {
			require.Equal(t, nomenclatureID, createReq.NomenclatureID)
			require.Equal(t, idempotencyKey, createReq.IdempotencyKey)
			require.Equal(t, models.DocumentTypeLetter, createReq.DocumentTypeID)
			require.Equal(t, deps.user.ID, createReq.CreatedBy)
			require.Equal(t, "12/26", createReq.IncomingNumber)
			require.Equal(t, "Incoming letter content", createReq.Content)
			require.Equal(t, 3, createReq.PagesCount)
			require.NotNil(t, createReq.Resolution)
			require.NotNil(t, createReq.ResolutionAuthor)
			require.NotNil(t, createReq.ResolutionExecutors)
			require.Len(t, createReq.Correspondents, 1)
			correspondent := createReq.Correspondents[0]
			return correspondent.RegistrationNumber == "A-1" &&
				correspondent.CorrespondentOrgID == orgID &&
				correspondent.Position == 1 &&
				createReq.IncomingDate.Equal(time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC))
		})).Return(&models.IncomingDocument{
			ID:             documentID,
			NomenclatureID: nomenclatureID,
			IncomingNumber: "12/26",
			IncomingDate:   time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
			DocumentTypeID: models.DocumentTypeLetter,
			Content:        req.Content,
			PagesCount:     req.PagesCount,
			CreatedBy:      deps.user.ID,
		}, nil).Once()
		deps.journalRepo.On("Create", context.Background(), mock.MatchedBy(func(journalReq models.CreateJournalEntryRequest) bool {
			return journalReq.DocumentID == documentID &&
				journalReq.UserID == deps.user.ID &&
				journalReq.Action == "CREATE" &&
				journalReq.Details == "Документ зарегистрирован. Рег. номер: 12/26"
		})).Return(uuid.New(), nil).Once()

		result, err := deps.handler.Register(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, documentID.String(), result.ID)
		assert.Equal(t, "12/26", result.IncomingNumber)
	})

	t.Run("rejects missing create permission", func(t *testing.T) {
		deps := setupIncomingLetterCommandHandler(t, nil)
		req := validIncomingLetterRegisterRequest(uuid.New(), uuid.New())

		result, err := deps.handler.Register(req)

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})

	t.Run("rejects invalid idempotency key", func(t *testing.T) {
		deps := setupIncomingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindIncomingLetter, "create"),
		)
		req := validIncomingLetterRegisterRequest(uuid.New(), uuid.New())
		req.IdempotencyKey = uuid.Nil.String()

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный ключ идемпотентности")
		assert.Nil(t, result)
	})

	t.Run("rejects invalid document type", func(t *testing.T) {
		deps := setupIncomingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindIncomingLetter, "create"),
		)
		req := validIncomingLetterRegisterRequest(uuid.New(), uuid.New())
		req.DocumentTypeID = "Неизвестный тип"

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный тип документа")
		assert.Nil(t, result)
	})

	t.Run("rejects invalid correspondent date", func(t *testing.T) {
		deps := setupIncomingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindIncomingLetter, "create"),
		)
		req := validIncomingLetterRegisterRequest(uuid.New(), uuid.New())
		req.Correspondents[0].RegistrationDate = "03.06.2026"

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный формат даты регистрации корреспондента")
		assert.Nil(t, result)
	})
}

func TestIncomingLetterCommandHandler_Update(t *testing.T) {
	t.Run("updates incoming letter and writes journal entry", func(t *testing.T) {
		documentID := uuid.New()
		orgID := uuid.New()
		deps := setupIncomingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindIncomingLetter, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindIncomingLetter),
			},
		}
		req := IncomingLetterUpdateRequest{
			ID:              documentID.String(),
			DocumentTypeID:  models.DocumentTypeLetter,
			Content:         "Updated content",
			PagesCount:      5,
			SenderSignatory: "Updated sender",
			Correspondents: []IncomingLetterCorrespondentRequest{
				{
					RegistrationNumber: "B-2",
					RegistrationDate:   "2026-06-01",
					CorrespondentName:  "АО Василек",
				},
			},
		}

		deps.refRepo.On("FindOrCreateOrganization", "АО Василек").Return(&models.Organization{ID: orgID, Name: "АО Василек"}, nil).Once()
		deps.repo.On("Update", mock.MatchedBy(func(updateReq models.UpdateIncomingDocRequest) bool {
			require.Equal(t, documentID, updateReq.ID)
			require.Equal(t, models.DocumentTypeLetter, updateReq.DocumentTypeID)
			require.Equal(t, "Updated content", updateReq.Content)
			require.Equal(t, 5, updateReq.PagesCount)
			require.Len(t, updateReq.Correspondents, 1)
			correspondent := updateReq.Correspondents[0]
			return correspondent.RegistrationNumber == "B-2" &&
				correspondent.CorrespondentOrgID == orgID &&
				correspondent.Position == 1
		})).Return(&models.IncomingDocument{
			ID:             documentID,
			IncomingNumber: "12/26",
			DocumentTypeID: models.DocumentTypeLetter,
			Content:        req.Content,
			PagesCount:     req.PagesCount,
			CreatedBy:      deps.user.ID,
		}, nil).Once()
		deps.journalRepo.On("Create", context.Background(), mock.MatchedBy(func(journalReq models.CreateJournalEntryRequest) bool {
			return journalReq.DocumentID == documentID &&
				journalReq.UserID == deps.user.ID &&
				journalReq.Action == "UPDATE" &&
				journalReq.Details == "Документ отредактирован"
		})).Return(uuid.New(), nil).Once()

		result, err := deps.handler.Update(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, documentID.String(), result.ID)
		assert.Equal(t, "Updated content", result.Content)
	})

	t.Run("rejects invalid document ID", func(t *testing.T) {
		deps := setupIncomingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindIncomingLetter, "read", "update"),
		)

		result, err := deps.handler.Update(IncomingLetterUpdateRequest{ID: "bad-id"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный ID документа")
		assert.Nil(t, result)
	})

	t.Run("rejects missing update permission", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupIncomingLetterCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindIncomingLetter),
			},
		}

		result, err := deps.handler.Update(IncomingLetterUpdateRequest{
			ID:             documentID.String(),
			DocumentTypeID: models.DocumentTypeLetter,
			Correspondents: []IncomingLetterCorrespondentRequest{
				{
					RegistrationNumber: "B-2",
					RegistrationDate:   "2026-06-01",
					CorrespondentName:  "АО Василек",
				},
			},
		})

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

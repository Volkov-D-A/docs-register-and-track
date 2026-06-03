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

type citizenAppealHandlerDeps struct {
	handler     *CitizenAppealCommandHandler
	repo        *citizenAppealCommandStore
	refRepo     *mocks.ReferenceStore
	journalRepo *mocks.JournalStore
	auth        *AuthService
	user        *models.User
}

type citizenAppealCommandStore struct {
	createReq    *models.CreateCitizenAppealDocRequest
	createResult *models.CitizenAppealDocument
	createErr    error
	updateReq    *models.UpdateCitizenAppealDocRequest
	updateResult *models.CitizenAppealDocument
	updateErr    error
}

func (s *citizenAppealCommandStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.CitizenAppealDocument], error) {
	return nil, nil
}

func (s *citizenAppealCommandStore) GetByID(id uuid.UUID) (*models.CitizenAppealDocument, error) {
	return nil, nil
}

func (s *citizenAppealCommandStore) Create(req models.CreateCitizenAppealDocRequest) (*models.CitizenAppealDocument, error) {
	s.createReq = &req
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createResult != nil {
		return s.createResult, nil
	}
	return &models.CitizenAppealDocument{ID: uuid.New()}, nil
}

func (s *citizenAppealCommandStore) Update(req models.UpdateCitizenAppealDocRequest) (*models.CitizenAppealDocument, error) {
	s.updateReq = &req
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	if s.updateResult != nil {
		return s.updateResult, nil
	}
	return &models.CitizenAppealDocument{ID: req.ID}, nil
}

func (s *citizenAppealCommandStore) GetCount() (int, error) {
	return 0, nil
}

func setupCitizenAppealCommandHandler(t *testing.T, allowed map[models.DocumentKind]map[string]bool) *citizenAppealHandlerDeps {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	user := documentAccessUser(false, nil)
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	repo := &citizenAppealCommandStore{}
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
	handler := NewCitizenAppealCommandHandler(repo, nomRepo, refRepo, auth, journal, access)

	return &citizenAppealHandlerDeps{
		handler:     handler,
		repo:        repo,
		refRepo:     refRepo,
		journalRepo: journalRepo,
		auth:        auth,
		user:        user,
	}
}

func validCitizenAppealRegisterRequest(nomenclatureID, idempotencyKey uuid.UUID) CitizenAppealRegisterRequest {
	return CitizenAppealRegisterRequest{
		NomenclatureID:       nomenclatureID.String(),
		IdempotencyKey:       idempotencyKey.String(),
		RegistrationDate:     "2026-06-03",
		AppealDate:           "2026-06-02",
		ApplicantFullName:    " Иван Иванов ",
		RegistrationAddress:  " ул. Ленина, 1 ",
		AppealType:           " Жалоба ",
		ApplicantCategory:    " гражданин ",
		AppealPagesCount:     2,
		AttachmentPagesCount: 1,
		HasEnvelope:          true,
		ReceivedFromPOS:      true,
		Content:              " Содержание обращения ",
		RegistrationNumber:   " CA-10 ",
		Correspondents: []CitizenAppealCorrespondentRequest{
			{
				RegistrationNumber: "EXT-1",
				RegistrationDate:   "2026-06-01",
				CorrespondentName:  "Администрация",
			},
		},
		Resolutions: []CitizenAppealResolutionRequest{
			{
				Resolution:          " Подготовить ответ ",
				ResolutionAuthor:    " Руководитель ",
				ResolutionExecutors: "Исполнитель 1; Исполнитель 2",
			},
		},
	}
}

func TestCitizenAppealCommandHandler_Register(t *testing.T) {
	t.Run("creates appeal and writes journal entry", func(t *testing.T) {
		nomenclatureID := uuid.New()
		idempotencyKey := uuid.New()
		documentID := uuid.New()
		correspondentOrgID := uuid.New()
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "create"),
		)
		req := validCitizenAppealRegisterRequest(nomenclatureID, idempotencyKey)
		deps.repo.createResult = &models.CitizenAppealDocument{
			ID:                   documentID,
			NomenclatureID:       nomenclatureID,
			RegistrationNumber:   "CA-10",
			RegistrationDate:     time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
			AppealDate:           time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
			ApplicantFullName:    "Иван Иванов",
			RegistrationAddress:  "ул. Ленина, 1",
			AppealType:           AppealTypeComplaint,
			ApplicantCategory:    "гражданин",
			Content:              "Содержание обращения",
			AppealPagesCount:     2,
			AttachmentPagesCount: 1,
			HasEnvelope:          true,
			ReceivedFromPOS:      true,
			CreatedBy:            deps.user.ID,
		}

		deps.refRepo.On("FindOrCreateOrganization", "Администрация").Return(&models.Organization{ID: correspondentOrgID, Name: "Администрация"}, nil).Once()
		deps.refRepo.On("FindOrCreateResolutionExecutor", "Исполнитель 1").Return(&models.ResolutionExecutor{ID: uuid.New(), Name: "Исполнитель 1"}, nil).Once()
		deps.refRepo.On("FindOrCreateResolutionExecutor", "Исполнитель 2").Return(&models.ResolutionExecutor{ID: uuid.New(), Name: "Исполнитель 2"}, nil).Once()
		deps.journalRepo.On("Create", context.Background(), mock.MatchedBy(func(journalReq models.CreateJournalEntryRequest) bool {
			return journalReq.DocumentID == documentID &&
				journalReq.UserID == deps.user.ID &&
				journalReq.Action == "CREATE" &&
				journalReq.Details == "Обращение зарегистрировано. Номер: CA-10"
		})).Return(uuid.New(), nil).Once()

		result, err := deps.handler.Register(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, deps.repo.createReq)
		assert.Equal(t, documentID.String(), result.ID)
		assert.Equal(t, nomenclatureID, deps.repo.createReq.NomenclatureID)
		assert.Equal(t, idempotencyKey, deps.repo.createReq.IdempotencyKey)
		assert.Equal(t, deps.user.ID, deps.repo.createReq.CreatedBy)
		assert.Equal(t, "CA-10", deps.repo.createReq.RegistrationNumber)
		assert.Equal(t, AppealTypeComplaint, deps.repo.createReq.AppealType)
		assert.Equal(t, "Иван Иванов", deps.repo.createReq.ApplicantFullName)
		assert.Equal(t, "ул. Ленина, 1", deps.repo.createReq.RegistrationAddress)
		assert.Equal(t, "гражданин", deps.repo.createReq.ApplicantCategory)
		assert.Equal(t, "Содержание обращения", deps.repo.createReq.Content)
		assert.Len(t, deps.repo.createReq.Correspondents, 1)
		assert.Equal(t, correspondentOrgID, deps.repo.createReq.Correspondents[0].CorrespondentOrgID)
		assert.Len(t, deps.repo.createReq.Resolutions, 1)
		assert.Equal(t, "Подготовить ответ", *deps.repo.createReq.Resolutions[0].Resolution)
		assert.Equal(t, "Руководитель", *deps.repo.createReq.Resolutions[0].ResolutionAuthor)
		assert.Equal(t, "Исполнитель 1; Исполнитель 2", *deps.repo.createReq.Resolutions[0].ResolutionExecutors)
	})

	t.Run("rejects missing create permission", func(t *testing.T) {
		deps := setupCitizenAppealCommandHandler(t, nil)
		req := validCitizenAppealRegisterRequest(uuid.New(), uuid.New())

		result, err := deps.handler.Register(req)

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})

	t.Run("rejects unsupported appeal type", func(t *testing.T) {
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "create"),
		)
		req := validCitizenAppealRegisterRequest(uuid.New(), uuid.New())
		req.AppealType = "запрос"

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный вид обращения")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})

	t.Run("rejects empty required applicant fields", func(t *testing.T) {
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "create"),
		)
		req := validCitizenAppealRegisterRequest(uuid.New(), uuid.New())
		req.ApplicantFullName = "  "

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "укажите ФИО обратившегося")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})

	t.Run("rejects negative attachment pages", func(t *testing.T) {
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "create"),
		)
		req := validCitizenAppealRegisterRequest(uuid.New(), uuid.New())
		req.AttachmentPagesCount = -1

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "количество листов приложения не может быть отрицательным")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})

	t.Run("rejects resolution without text", func(t *testing.T) {
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "create"),
		)
		req := validCitizenAppealRegisterRequest(uuid.New(), uuid.New())
		req.Correspondents = nil
		req.Resolutions = []CitizenAppealResolutionRequest{{ResolutionAuthor: "Руководитель"}}

		result, err := deps.handler.Register(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "укажите текст резолюции")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.createReq)
	})
}

func TestCitizenAppealCommandHandler_Update(t *testing.T) {
	t.Run("updates appeal and writes journal entry", func(t *testing.T) {
		documentID := uuid.New()
		correspondentOrgID := uuid.New()
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindCitizenAppeal),
			},
		}
		deps.repo.updateResult = &models.CitizenAppealDocument{
			ID:                   documentID,
			RegistrationNumber:   "CA-20",
			RegistrationDate:     time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
			AppealDate:           time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
			ApplicantFullName:    "Петр Петров",
			RegistrationAddress:  "ул. Мира, 2",
			AppealType:           AppealTypeApplication,
			ApplicantCategory:    "пенсионер",
			Content:              "Обновленное обращение",
			AppealPagesCount:     3,
			AttachmentPagesCount: 0,
			CreatedBy:            deps.user.ID,
		}
		req := CitizenAppealUpdateRequest{
			ID:                   documentID.String(),
			RegistrationNumber:   " CA-20 ",
			RegistrationDate:     "2026-06-04",
			AppealDate:           "2026-06-03",
			ApplicantFullName:    " Петр Петров ",
			RegistrationAddress:  " ул. Мира, 2 ",
			AppealType:           "ЗАЯВЛЕНИЕ",
			ApplicantCategory:    " пенсионер ",
			AppealPagesCount:     3,
			AttachmentPagesCount: 0,
			Content:              " Обновленное обращение ",
			Correspondents: []CitizenAppealCorrespondentRequest{
				{
					RegistrationNumber: "EXT-2",
					RegistrationDate:   "2026-06-02",
					CorrespondentName:  "Прокуратура",
				},
			},
		}

		deps.refRepo.On("FindOrCreateOrganization", "Прокуратура").Return(&models.Organization{ID: correspondentOrgID, Name: "Прокуратура"}, nil).Once()
		deps.journalRepo.On("Create", context.Background(), mock.MatchedBy(func(journalReq models.CreateJournalEntryRequest) bool {
			return journalReq.DocumentID == documentID &&
				journalReq.UserID == deps.user.ID &&
				journalReq.Action == "UPDATE" &&
				journalReq.Details == "Обращение отредактировано"
		})).Return(uuid.New(), nil).Once()

		result, err := deps.handler.Update(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, deps.repo.updateReq)
		assert.Equal(t, documentID.String(), result.ID)
		assert.Equal(t, documentID, deps.repo.updateReq.ID)
		assert.Equal(t, "CA-20", deps.repo.updateReq.RegistrationNumber)
		assert.Equal(t, AppealTypeApplication, deps.repo.updateReq.AppealType)
		assert.Equal(t, "Петр Петров", deps.repo.updateReq.ApplicantFullName)
		assert.Equal(t, "Обновленное обращение", deps.repo.updateReq.Content)
		assert.Len(t, deps.repo.updateReq.Correspondents, 1)
		assert.Equal(t, correspondentOrgID, deps.repo.updateReq.Correspondents[0].CorrespondentOrgID)
	})

	t.Run("rejects invalid document ID", func(t *testing.T) {
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "read", "update"),
		)

		result, err := deps.handler.Update(CitizenAppealUpdateRequest{ID: "bad-id"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неверный ID документа")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.updateReq)
	})

	t.Run("rejects missing update permission", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "read"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindCitizenAppeal),
			},
		}

		result, err := deps.handler.Update(CitizenAppealUpdateRequest{ID: documentID.String()})

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.updateReq)
	})

	t.Run("rejects empty registration number", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupCitizenAppealCommandHandler(
			t,
			allowDocumentActions(models.DocumentKindCitizenAppeal, "read", "update"),
		)
		deps.handler.access.documentRepo = &documentAccessDocumentStore{
			docs: map[uuid.UUID]models.Document{
				documentID: documentAccessDoc(documentID, uuid.New(), models.DocumentKindCitizenAppeal),
			},
		}

		result, err := deps.handler.Update(CitizenAppealUpdateRequest{
			ID:                 documentID.String(),
			RegistrationNumber: "  ",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "укажите номер документа")
		assert.Nil(t, result)
		assert.Nil(t, deps.repo.updateReq)
	})
}

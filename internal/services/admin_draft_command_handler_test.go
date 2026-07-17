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

func validAdminDraftCreateRequest(nomenclatureID uuid.UUID) AdminDraftCreateRequest {
	return AdminDraftCreateRequest{
		NomenclatureID:   nomenclatureID.String(),
		RegistrationDate: "2026-06-08",
		AdminNumberOverride: &AdminNumberOverrideRequest{
			Mode:   models.AdminNumberModeLiteral,
			Number: 15,
			Suffix: "А",
		},
	}
}

func assertAdminDraftOverride(t *testing.T, override *models.AdminNumberOverride) {
	t.Helper()

	require.NotNil(t, override)
	assert.Equal(t, models.AdminNumberModeLiteral, override.Mode)
	assert.Equal(t, 15, override.Number)
	assert.Equal(t, "А", override.Suffix)
}

func expectAdminDraftJournal(t *testing.T, repo *mocks.JournalStore, documentID, userID uuid.UUID, number string) {
	t.Helper()

	repo.On("Create", context.Background(), mock.MatchedBy(func(req models.CreateJournalEntryRequest) bool {
		return req.DocumentID == documentID &&
			req.UserID == userID &&
			req.Action == "ADMIN_DRAFT_CREATE" &&
			req.Details == "Создан административный черновик. Рег. номер: "+number
	})).Return(uuid.New(), nil).Once()
}

func TestIncomingLetterCommandHandler_CreateAdminDraft(t *testing.T) {
	t.Run("creates admin draft with placeholder fields", func(t *testing.T) {
		nomenclatureID := uuid.New()
		orgID := uuid.New()
		documentID := uuid.New()
		registrationDate := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
		deps := setupIncomingLetterCommandHandler(t, nil)
		deps.auth.SetAccessStore(newRoleMappedDocumentAccessStore(models.SystemPermissionAdmin))

		deps.refRepo.On("FindOrCreateOrganization", adminDraftPlaceholder).Return(&models.Organization{ID: orgID, Name: adminDraftPlaceholder}, nil).Once()
		deps.repo.On("Create", mock.MatchedBy(func(req models.CreateIncomingDocRequest) bool {
			assert.Equal(t, nomenclatureID, req.NomenclatureID)
			assert.NotEqual(t, uuid.Nil, req.IdempotencyKey)
			assert.Equal(t, models.DocumentTypeLetter, req.DocumentTypeID)
			assert.Equal(t, deps.user.ID, req.CreatedBy)
			assert.Equal(t, registrationDate, req.IncomingDate)
			assert.Equal(t, adminDraftPlaceholder, req.Content)
			assert.Equal(t, 1, req.PagesCount)
			assert.Equal(t, adminDraftPlaceholder, req.SenderSignatory)
			require.Len(t, req.Correspondents, 1)
			assert.Equal(t, adminDraftPlaceholder, req.Correspondents[0].RegistrationNumber)
			assert.Equal(t, registrationDate, req.Correspondents[0].RegistrationDate)
			assert.Equal(t, orgID, req.Correspondents[0].CorrespondentOrgID)
			assertAdminDraftOverride(t, req.AdminNumberOverride)
			return true
		})).Return(&models.IncomingDocument{
			ID:             documentID,
			NomenclatureID: nomenclatureID,
			IncomingNumber: "26-01-27/15А",
			IncomingDate:   registrationDate,
			DocumentTypeID: models.DocumentTypeLetter,
			Content:        adminDraftPlaceholder,
			PagesCount:     1,
			CreatedBy:      deps.user.ID,
		}, nil).Once()
		result, err := deps.handler.CreateAdminDraft(validAdminDraftCreateRequest(nomenclatureID))

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("rejects missing admin permission", func(t *testing.T) {
		deps := setupIncomingLetterCommandHandler(t, nil)

		result, err := deps.handler.CreateAdminDraft(validAdminDraftCreateRequest(uuid.New()))

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

func TestOutgoingLetterCommandHandler_CreateAdminDraft(t *testing.T) {
	nomenclatureID := uuid.New()
	orgID := uuid.New()
	documentID := uuid.New()
	registrationDate := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	deps := setupOutgoingLetterCommandHandler(t, nil)
	deps.auth.SetAccessStore(newRoleMappedDocumentAccessStore(models.SystemPermissionAdmin))

	deps.refRepo.On("FindOrCreateOrganization", adminDraftPlaceholder).Return(&models.Organization{ID: orgID, Name: adminDraftPlaceholder}, nil).Once()
	deps.repo.On("Create", mock.MatchedBy(func(req models.CreateOutgoingDocRequest) bool {
		assert.Equal(t, nomenclatureID, req.NomenclatureID)
		assert.NotEqual(t, uuid.Nil, req.IdempotencyKey)
		assert.Equal(t, models.DocumentTypeLetter, req.DocumentTypeID)
		assert.Equal(t, orgID, req.RecipientOrgID)
		assert.Equal(t, deps.user.ID, req.CreatedBy)
		assert.Equal(t, registrationDate, req.OutgoingDate)
		assert.Equal(t, adminDraftPlaceholder, req.Content)
		assert.Equal(t, 1, req.PagesCount)
		assert.Equal(t, adminDraftPlaceholder, req.SenderSignatory)
		assert.Equal(t, adminDraftPlaceholder, req.SenderExecutor)
		assert.Equal(t, adminDraftPlaceholder, req.Addressee)
		assertAdminDraftOverride(t, req.AdminNumberOverride)
		return true
	})).Return(&models.OutgoingDocument{
		ID:             documentID,
		NomenclatureID: nomenclatureID,
		OutgoingNumber: "26-01-27/15А",
		OutgoingDate:   registrationDate,
		DocumentTypeID: models.DocumentTypeLetter,
		Content:        adminDraftPlaceholder,
		PagesCount:     1,
		CreatedBy:      deps.user.ID,
	}, nil).Once()
	result, err := deps.handler.CreateAdminDraft(validAdminDraftCreateRequest(nomenclatureID))

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestCitizenAppealCommandHandler_CreateAdminDraft(t *testing.T) {
	nomenclatureID := uuid.New()
	documentID := uuid.New()
	registrationDate := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	deps := setupCitizenAppealCommandHandler(t, nil)
	deps.auth.SetAccessStore(newRoleMappedDocumentAccessStore(models.SystemPermissionAdmin))
	deps.repo.createResult = &models.CitizenAppealDocument{
		ID:                 documentID,
		NomenclatureID:     nomenclatureID,
		RegistrationNumber: "26-01-27/15А",
		RegistrationDate:   registrationDate,
		AppealDate:         registrationDate,
		Content:            adminDraftPlaceholder,
		CreatedBy:          deps.user.ID,
	}
	result, err := deps.handler.CreateAdminDraft(validAdminDraftCreateRequest(nomenclatureID))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, deps.repo.createReq)
	req := deps.repo.createReq
	assert.Equal(t, nomenclatureID, req.NomenclatureID)
	assert.NotEqual(t, uuid.Nil, req.IdempotencyKey)
	assert.Equal(t, deps.user.ID, req.CreatedBy)
	assert.Equal(t, registrationDate, req.RegistrationDate)
	assert.Equal(t, registrationDate, req.AppealDate)
	assert.Equal(t, adminDraftPlaceholder, req.Content)
	assert.Equal(t, adminDraftPlaceholder, req.ApplicantFullName)
	assert.Equal(t, adminDraftPlaceholder, req.RegistrationAddress)
	assert.Equal(t, adminDraftPlaceholder, req.ApplicantCategory)
	assert.Equal(t, AppealTypeApplication, req.AppealType)
	assert.Equal(t, 1, req.AppealPagesCount)
	assert.Equal(t, 0, req.AttachmentPagesCount)
	assertAdminDraftOverride(t, req.AdminNumberOverride)
}

func TestAdministrativeOrderCommandHandler_CreateAdminDraft(t *testing.T) {
	nomenclatureID := uuid.New()
	documentID := uuid.New()
	registrationDate := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	deps := setupAdministrativeOrderCommandHandler(t, nil)
	deps.auth.SetAccessStore(newRoleMappedDocumentAccessStore(models.SystemPermissionAdmin))
	deps.repo.createResult = &models.AdministrativeOrderDocument{
		ID:                  documentID,
		NomenclatureID:      nomenclatureID,
		OrderNumber:         "26-01-27/15А",
		OrderDate:           registrationDate,
		Title:               adminDraftPlaceholder,
		ExecutionController: adminDraftPlaceholder,
		IsActive:            true,
		CreatedBy:           deps.user.ID,
	}
	result, err := deps.handler.CreateAdminDraft(validAdminDraftCreateRequest(nomenclatureID))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, deps.repo.createReq)
	req := deps.repo.createReq
	assert.Equal(t, nomenclatureID, req.NomenclatureID)
	assert.NotEqual(t, uuid.Nil, req.IdempotencyKey)
	assert.Equal(t, deps.user.ID, req.CreatedBy)
	assert.Equal(t, registrationDate, req.OrderDate)
	assert.Equal(t, adminDraftPlaceholder, req.Title)
	assert.Equal(t, adminDraftPlaceholder, req.ExecutionController)
	assert.True(t, req.IsActive)
	assert.Empty(t, req.AcknowledgmentFullNames)
	assertAdminDraftOverride(t, req.AdminNumberOverride)
}

type documentRegistrationServiceDraftHandler struct {
	kind models.DocumentKind
	req  *AdminDraftCreateRequest
}

func (h *documentRegistrationServiceDraftHandler) Kind() models.DocumentKind {
	return h.kind
}

func (h *documentRegistrationServiceDraftHandler) RegisterDocument(req any) (any, error) {
	return nil, models.ErrForbidden
}

func (h *documentRegistrationServiceDraftHandler) UpdateDocument(req any) (any, error) {
	return nil, models.ErrForbidden
}

func (h *documentRegistrationServiceDraftHandler) CreateAdminDraft(req AdminDraftCreateRequest) (any, error) {
	h.req = &req
	return "created", nil
}

type documentRegistrationServicePlainHandler struct {
	kind models.DocumentKind
}

func (h *documentRegistrationServicePlainHandler) Kind() models.DocumentKind {
	return h.kind
}

func (h *documentRegistrationServicePlainHandler) RegisterDocument(req any) (any, error) {
	return nil, models.ErrForbidden
}

func (h *documentRegistrationServicePlainHandler) UpdateDocument(req any) (any, error) {
	return nil, models.ErrForbidden
}

func TestDocumentRegistrationService_CreateAdminDraft(t *testing.T) {
	t.Run("delegates to admin draft handler", func(t *testing.T) {
		handler := &documentRegistrationServiceDraftHandler{kind: models.DocumentKindIncomingLetter}
		service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry(handler))
		req := validAdminDraftCreateRequest(uuid.New())

		result, err := service.CreateAdminDraft(string(models.DocumentKindIncomingLetter), req)

		require.NoError(t, err)
		assert.Equal(t, "created", result)
		require.NotNil(t, handler.req)
		assert.Equal(t, req, *handler.req)
	})

	t.Run("rejects handler without admin draft support", func(t *testing.T) {
		handler := &documentRegistrationServicePlainHandler{kind: models.DocumentKindIncomingLetter}
		service := NewDocumentRegistrationService(NewDocumentKindCommandRegistry(handler))

		result, err := service.CreateAdminDraft(string(models.DocumentKindIncomingLetter), validAdminDraftCreateRequest(uuid.New()))

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

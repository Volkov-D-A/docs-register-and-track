package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupOutgoingDocService(t *testing.T, role string) (
	*OutgoingDocumentService, *mocks.OutgoingDocStore, *mocks.ReferenceStore, *mocks.NomenclatureStore, *mocks.DepartmentStore, *mocks.SettingsStore, *AuthService,
) {
	t.Helper()
	outRepo := mocks.NewOutgoingDocStore(t)
	refRepo := mocks.NewReferenceStore(t)
	nomRepo := mocks.NewNomenclatureStore(t)
	depRepo := mocks.NewDepartmentStore(t)
	settingsRepo := mocks.NewSettingsStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_out",
		PasswordHash: hash,
		FullName:     "Test User",
		IsActive:     true,
		Roles:        []string{role},
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	settingsSvc := NewSettingsService(nil, settingsRepo, auth, nil)
	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth)

	svc := NewOutgoingDocumentService(outRepo, refRepo, nomRepo, depRepo, auth, settingsSvc, journalSvc)
	return svc, outRepo, refRepo, nomRepo, depRepo, settingsRepo, auth
}

func TestOutgoingDocService_Register(t *testing.T) {
	// Регистрация нового исходящего документа
	nomID := uuid.New()
	docTypeID := uuid.New()
	recipientOrg := &models.Organization{ID: uuid.New(), Name: "Получатель"}
	senderOrg := &models.Organization{ID: uuid.New(), Name: "НАША ОРГАНИЗАЦИЯ"}

	t.Run("success clerk", func(t *testing.T) {
		svc, outRepo, refRepo, nomRepo, _, settingsRepo, _ := setupOutgoingDocService(t, "clerk")

		refRepo.On("FindOrCreateOrganization", "Получатель").Return(recipientOrg, nil).Once()
		settingsRepo.On("Get", "organization_name").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		refRepo.On("FindOrCreateOrganization", "НАША ОРГАНИЗАЦИЯ").Return(senderOrg, nil).Once()
		nomRepo.On("GetNextNumber", nomID).Return(1, "01-01", nil).Once()
		outRepo.On("Create", mock.AnythingOfType("models.CreateOutgoingDocRequest")).Return(&models.OutgoingDocument{
			ID: uuid.New(),
		}, nil).Once()

		result, err := svc.Register(nomID.String(), docTypeID.String(), "Получатель", "Директору", "2025-06-15", "Тема", "Контент", 3, "Подписант", "Исполнитель")
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _, _, _, _, _, _ := setupOutgoingDocService(t, "executor")
		result, err := svc.Register(nomID.String(), docTypeID.String(), "Получатель", "Директору", "2025-06-15", "Тема", "Контент", 3, "Подписант", "Исполнитель")
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("invalid nomenclature", func(t *testing.T) {
		svc, _, _, _, _, _, _ := setupOutgoingDocService(t, "clerk")
		result, err := svc.Register("not-uuid", docTypeID.String(), "Получатель", "Директору", "2025-06-15", "Тема", "Контент", 3, "Подписант", "Исполнитель")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "номенклатуры")
		assert.Nil(t, result)
	})
}

func TestOutgoingDocService_Update(t *testing.T) {
	// Обновление карточки исходящего документа
	docID := uuid.New()
	docTypeID := uuid.New()
	recipientOrg := &models.Organization{ID: uuid.New(), Name: "Получатель"}
	senderOrg := &models.Organization{ID: uuid.New(), Name: "НАША ОРГАНИЗАЦИЯ"}

	t.Run("success", func(t *testing.T) {
		svc, outRepo, refRepo, _, _, settingsRepo, _ := setupOutgoingDocService(t, "clerk")

		refRepo.On("FindOrCreateOrganization", "Получатель").Return(recipientOrg, nil).Once()
		settingsRepo.On("Get", "organization_name").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		refRepo.On("FindOrCreateOrganization", "НАША ОРГАНИЗАЦИЯ").Return(senderOrg, nil).Once()
		outRepo.On("Update", mock.AnythingOfType("models.UpdateOutgoingDocRequest")).Return(&models.OutgoingDocument{
			ID: docID,
		}, nil).Once()

		result, err := svc.Update(docID.String(), docTypeID.String(), "Получатель", "Директору", "2025-06-15", "Тема", "Контент", 3, "Подписант", "Исполнитель")
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, _, _, _, _, _, _ := setupOutgoingDocService(t, "executor")
		result, err := svc.Update(docID.String(), docTypeID.String(), "Получатель", "Директору", "2025-06-15", "Тема", "Контент", 3, "Подписант", "Исполнитель")
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _, _, _, _, _, _ := setupOutgoingDocService(t, "clerk")
		result, err := svc.Update("not-uuid", docTypeID.String(), "Получатель", "Директору", "2025-06-15", "Тема", "Контент", 3, "Подписант", "Исполнитель")
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestOutgoingDocService_GetByID(t *testing.T) {
	// Получение детальной информации об исходящем документе по ID
	docID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, outRepo, _, _, _, _, _ := setupOutgoingDocService(t, "executor")
		outRepo.On("GetByID", docID).Return(&models.OutgoingDocument{ID: docID, Subject: "Тема"}, nil).Once()
		result, err := svc.GetByID(docID.String())
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, docID.String(), result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		svc, outRepo, _, _, _, _, _ := setupOutgoingDocService(t, "executor")
		outRepo.On("GetByID", docID).Return((*models.OutgoingDocument)(nil), nil).Once()
		result, err := svc.GetByID(docID.String())
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestOutgoingDocService_Delete(t *testing.T) {
	// Удаление исходящего документа из системы
	docID := uuid.New()

	t.Run("success admin", func(t *testing.T) {
		svc, outRepo, _, _, _, _, _ := setupOutgoingDocService(t, "admin")
		outRepo.On("Delete", docID).Return(nil).Once()
		err := svc.Delete(docID.String())
		require.NoError(t, err)
	})

	t.Run("forbidden clerk", func(t *testing.T) {
		svc, _, _, _, _, _, _ := setupOutgoingDocService(t, "clerk")
		err := svc.Delete(docID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})
}

func TestOutgoingDocService_GetCount(t *testing.T) {
	// Подсчет общего количества зарегистрированных исходящих документов
	t.Run("success", func(t *testing.T) {
		svc, outRepo, _, _, _, _, _ := setupOutgoingDocService(t, "executor")
		outRepo.On("GetCount").Return(42, nil).Once()
		count, err := svc.GetCount()
		require.NoError(t, err)
		assert.Equal(t, 42, count)
	})
}

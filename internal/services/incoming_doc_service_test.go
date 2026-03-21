package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIncomingDocumentService_Register(t *testing.T) {
	// Регистрация нового входящего документа (создание карточки и генерация номера)
	mockDocRepo := mocks.NewIncomingDocStore(t)
	mockNomRepo := mocks.NewNomenclatureStore(t)
	mockRefRepo := mocks.NewReferenceStore(t)
	mockDepRepo := mocks.NewDepartmentStore(t)

	authRepo := mocks.NewUserStore(t)
	authService := NewAuthService(nil, authRepo)

	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, authService)

	docService := NewIncomingDocumentService(mockDocRepo, mockNomRepo, mockRefRepo, mockDepRepo, authService, journalSvc)

	login := "testuser"
	password := "CorrectPassw0rd!"
	hash, _ := security.HashPassword(password)
	userID := uuid.New()

	clerkUser := &models.User{
		ID:           userID,
		Login:        login,
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{"clerk"},
	}

	executorUser := &models.User{
		ID:           uuid.New(),
		Login:        "executor",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{"executor"},
	}

	t.Run("успех делопроизводитель", func(t *testing.T) {
		authRepo.On("GetByLogin", login).Return(clerkUser, nil).Once()
		authService.Login(login, password)
		authRepo.On("GetByID", clerkUser.ID).Return(clerkUser, nil).Maybe()

		nomIDStr := uuid.New().String()
		docTypeIDStr := uuid.New().String()
		nomID, _ := uuid.Parse(nomIDStr)

		orgMap := &models.Organization{ID: uuid.New(), Name: "Org"}

		mockRefRepo.On("FindOrCreateOrganization", "Sender").Return(orgMap, nil).Once()
		mockRefRepo.On("FindOrCreateOrganization", "Recipient").Return(orgMap, nil).Once()
		mockNomRepo.On("GetNextNumber", nomID).Return(1, "01-01", nil).Once()

		expectedModel := &models.IncomingDocument{
			ID:             uuid.New(),
			IncomingNumber: "01-01/1",
		}

		mockDocRepo.On("Create",
			mock.AnythingOfType("models.CreateIncomingDocRequest"),
		).Return(expectedModel, nil).Once()

		doc, err := docService.Register(
			nomIDStr, docTypeIDStr, "Sender", "Recipient",
			"2024-01-01", "2024-01-01", "Out-123",
			"", "", "Subject", "Content", 1, "Signatory", "Executor", "Addressee", "Resolution",
		)

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, "01-01/1", doc.IncomingNumber)

		authService.Logout()
	})

	t.Run("запрещено исполнитель", func(t *testing.T) {
		authRepo.On("GetByLogin", "executor").Return(executorUser, nil).Once()
		authService.Login("executor", password)
		authRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()

		doc, err := docService.Register(
			uuid.New().String(), uuid.New().String(), "Sender", "Recipient",
			"2024-01-01", "2024-01-01", "", "", "", "", "", 1, "", "", "", "",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "недостаточно прав")
		assert.Nil(t, doc)

		authService.Logout()
	})
}

func setupIncomingDocService(t *testing.T, role string) (
	*IncomingDocumentService, *mocks.IncomingDocStore, *mocks.NomenclatureStore, *mocks.ReferenceStore, *mocks.DepartmentStore,
) {
	t.Helper()
	docRepo := mocks.NewIncomingDocStore(t)
	nomRepo := mocks.NewNomenclatureStore(t)
	refRepo := mocks.NewReferenceStore(t)
	depRepo := mocks.NewDepartmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_inc",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{role},
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	auth.Login(user.Login, password)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth)

	svc := NewIncomingDocumentService(docRepo, nomRepo, refRepo, depRepo, auth, journalSvc)
	return svc, docRepo, nomRepo, refRepo, depRepo
}

func TestIncomingDocumentService_GetByID(t *testing.T) {
	// Получение всей информации о входящем документе по его ID
	docID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _, _ := setupIncomingDocService(t, "executor")
		repo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, Subject: "Тема"}, nil).Once()
		result, err := svc.GetByID(docID.String())
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, docID.String(), result.ID)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _, _, _, _ := setupIncomingDocService(t, "executor")
		result, err := svc.GetByID("not-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID")
		assert.Nil(t, result)
	})
}

func TestIncomingDocumentService_Delete(t *testing.T) {
	// Удаление карточки входящего документа
	docID := uuid.New()

	t.Run("success admin", func(t *testing.T) {
		svc, repo, _, _, _ := setupIncomingDocService(t, "admin")
		repo.On("Delete", docID).Return(nil).Once()
		err := svc.Delete(docID.String())
		require.NoError(t, err)
	})

	t.Run("forbidden clerk", func(t *testing.T) {
		svc, _, _, _, _ := setupIncomingDocService(t, "clerk")
		err := svc.Delete(docID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})
}

func TestIncomingDocumentService_GetCount(t *testing.T) {
	// Получение общего количества зарегистрированных входящих документов
	t.Run("success", func(t *testing.T) {
		svc, repo, _, _, _ := setupIncomingDocService(t, "executor")
		repo.On("GetCount").Return(15, nil).Once()
		count, err := svc.GetCount()
		require.NoError(t, err)
		assert.Equal(t, 15, count)
	})
}

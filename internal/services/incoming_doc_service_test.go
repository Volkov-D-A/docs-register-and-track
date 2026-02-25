package services

import (
	"testing"

	"docflow/internal/mocks"
	"docflow/internal/models"
	"docflow/internal/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIncomingDocumentService_Register(t *testing.T) {
	mockDocRepo := mocks.NewIncomingDocStore(t)
	mockNomRepo := mocks.NewNomenclatureStore(t)
	mockRefRepo := mocks.NewReferenceStore(t)
	mockDepRepo := mocks.NewDepartmentStore(t)

	authRepo := mocks.NewUserStore(t)
	authService := NewAuthService(nil, authRepo)

	docService := NewIncomingDocumentService(mockDocRepo, mockNomRepo, mockRefRepo, mockDepRepo, authService)

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

		nomIDStr := uuid.New().String()
		docTypeIDStr := uuid.New().String()
		nomID, _ := uuid.Parse(nomIDStr)
		docTypeID, _ := uuid.Parse(docTypeIDStr)

		orgMap := &models.Organization{ID: uuid.New(), Name: "Org"}

		mockRefRepo.On("FindOrCreateOrganization", "Sender").Return(orgMap, nil).Once()
		mockRefRepo.On("FindOrCreateOrganization", "Recipient").Return(orgMap, nil).Once()
		mockNomRepo.On("GetNextNumber", nomID).Return(1, "01-01", nil).Once()

		expectedModel := &models.IncomingDocument{
			ID:             uuid.New(),
			IncomingNumber: "01-01/1",
		}

		mockDocRepo.On("Create",
			nomID, docTypeID, orgMap.ID, orgMap.ID, userID,
			"01-01/1", mock.AnythingOfType("time.Time"),
			"Out-123", mock.AnythingOfType("time.Time"),
			mock.AnythingOfType("*string"), mock.AnythingOfType("*time.Time"),
			"Subject", "Content", 1, "Signatory", "Executor", "Addressee",
			mock.AnythingOfType("*string"),
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

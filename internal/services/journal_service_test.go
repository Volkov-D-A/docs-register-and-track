package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupJournalService(t *testing.T, role string) (*JournalService, *mocks.JournalStore, *mocks.IncomingDocStore, *mocks.OutgoingDocStore, *AuthService) {
	t.Helper()
	journalRepo := mocks.NewJournalStore(t)
	incomingRepo := mocks.NewIncomingDocStore(t)
	outgoingRepo := mocks.NewOutgoingDocStore(t)
	depRepo := mocks.NewDepartmentStore(t)
	assignmentRepo := mocks.NewAssignmentStore(t)
	ackRepo := mocks.NewAcknowledgmentStore(t)
	userRepo := mocks.NewUserStore(t)

	auth := NewAuthService(nil, userRepo)

	if role != "" {
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		user := &models.User{
			ID:           uuid.New(),
			Login:        role + "_journal",
			PasswordHash: hash,
			IsActive:     true,
			Roles:        []string{role},
		}
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Maybe()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	}

	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, nil, incomingRepo, outgoingRepo)
	svc := NewJournalService(journalRepo, auth, accessSvc)
	return svc, journalRepo, incomingRepo, outgoingRepo, auth
}

func TestJournalService_GetByDocumentID(t *testing.T) {
	// Получение записей журнала для конкретного документа (вызывается фронтендом)
	docID := uuid.New()
	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _, _ := setupJournalService(t, "clerk")

		now := time.Now()
		mockEntries := []models.JournalEntry{
			{
				ID:         uuid.New(),
				DocumentID: docID,
				UserID:     uuid.New(),
				UserName:   "Иванов Иван Иванович",
				Action:     "TEST_ACTION",
				Details:    "Тестовое действие",
				CreatedAt:  now,
			},
		}

		repo.On("GetByDocumentID", mock.Anything, docID).Return(mockEntries, nil).Once()

		result, err := svc.GetByDocumentID(docID.String())
		require.NoError(t, err)
		assert.Len(t, result, 1)

		// Проверяем, что маппинг в DTO прошел корректно
		var expectedDTOs []dto.JournalEntry = dto.MapJournalEntries(mockEntries)
		assert.Equal(t, expectedDTOs, result)
		assert.Equal(t, "Иванов Иван Иванович", result[0].UserName)
		assert.Equal(t, "TEST_ACTION", result[0].Action)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _, _ := setupJournalService(t, "") // без авторизации

		result, err := svc.GetByDocumentID(docID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "требуется авторизация")
		assert.Nil(t, result)
	})

	t.Run("неверный ID", func(t *testing.T) {
		svc, _, _, _, _ := setupJournalService(t, "clerk")

		result, err := svc.GetByDocumentID("invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUID")
		assert.Nil(t, result)
	})

	t.Run("admin не имеет доступа к журналу документа", func(t *testing.T) {
		svc, _, _, _, _ := setupJournalService(t, "admin")

		result, err := svc.GetByDocumentID(docID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

func TestJournalService_LogAction(t *testing.T) {
	// Внутреннее логирование действия (вызывается другими сервисами)
	docID := uuid.New()
	userID := uuid.New()
	action := "TEST_ACTION"
	details := "Тестовые детали"

	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _, _ := setupJournalService(t, "admin")

		expectedReq := models.CreateJournalEntryRequest{
			DocumentID: docID,
			UserID:     userID,
			Action:     action,
			Details:    details,
		}

		repo.On("Create", mock.Anything, expectedReq).Return(uuid.New(), nil).Once()

		err := svc.LogAction(context.Background(), expectedReq)
		require.NoError(t, err)
	})
}

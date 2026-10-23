package services

import (
	"errors"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupJournalService(t *testing.T, documentID uuid.UUID, authenticated, visible bool, permissions ...string) (*JournalService, *mocks.JournalStore) {
	t.Helper()
	journalRepo := mocks.NewJournalStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := newDocumentPrincipalStub(userRepo)
	auth.systemPermissions = permissions
	if authenticated {
		user := &models.User{ID: uuid.New(), Login: "journal-user", IsActive: true}
		auth.currentUserID = user.ID
		userRepo.On("GetByID", user.ID).Return(user, nil)
	}
	documents := &documentAccessDocumentStore{docs: map[uuid.UUID]models.Document{
		documentID: {ID: documentID, Kind: models.DocumentKindIncomingLetter},
	}}
	accessStore := &kindActionDocumentAccessStore{visible: func(id uuid.UUID) bool { return visible && id == documentID }}
	return NewJournalService(journalRepo, NewDocumentAccessService(auth, accessStore, documents)), journalRepo
}

func TestJournalService_GetByDocumentID(t *testing.T) {
	// Получение записей журнала для конкретного документа (вызывается фронтендом)
	docID := uuid.New()
	t.Run("успех", func(t *testing.T) {
		svc, repo := setupJournalService(t, docID, true, true)

		now := time.Now()
		mockEntries := []models.JournalEntry{
			{
				ID:        uuid.New(),
				UserName:  "Иванов Иван Иванович",
				Action:    "TEST_ACTION",
				Details:   "Тестовое действие",
				CreatedAt: now,
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
		svc, _ := setupJournalService(t, docID, false, false) // без авторизации

		result, err := svc.GetByDocumentID(docID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "требуется авторизация")
		assert.Nil(t, result)
	})

	t.Run("неверный ID", func(t *testing.T) {
		svc, _ := setupJournalService(t, docID, false, false)

		result, err := svc.GetByDocumentID("invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUID")
		assert.Nil(t, result)
	})

	t.Run("admin не имеет доступа к журналу документа", func(t *testing.T) {
		svc, _ := setupJournalService(t, docID, true, false, models.SystemPermissionAdmin)

		result, err := svc.GetByDocumentID(docID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})

	t.Run("admin with document visibility can read journal", func(t *testing.T) {
		svc, repo := setupJournalService(t, docID, true, true, models.SystemPermissionAdmin)
		repo.On("GetByDocumentID", mock.Anything, docID).Return([]models.JournalEntry{}, nil).Once()
		result, err := svc.GetByDocumentID(docID.String())
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns repository error", func(t *testing.T) {
		svc, repo := setupJournalService(t, docID, true, true)
		expectedErr := errors.New("journal lookup failed")
		repo.On("GetByDocumentID", mock.Anything, docID).Return(nil, expectedErr).Once()

		result, err := svc.GetByDocumentID(docID.String())

		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
	})

	t.Run("denies reads when access service is absent", func(t *testing.T) {
		svc, _ := setupJournalService(t, docID, false, false)
		svc.access = nil

		result, err := svc.GetByDocumentID(docID.String())

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

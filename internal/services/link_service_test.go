package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

func setupLinkService(t *testing.T, role string) (*LinkService, *mocks.LinkStore, *mocks.IncomingDocStore, *mocks.OutgoingDocStore, *AuthService) {
	t.Helper()
	linkRepo := mocks.NewLinkStore(t)
	incRepo := mocks.NewIncomingDocStore(t)
	outRepo := mocks.NewOutgoingDocStore(t)
	userRepo := mocks.NewUserStore(t)

	auth := NewAuthService(nil, userRepo)

	if role != "" {
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		user := &models.User{
			ID:           uuid.New(),
			Login:        role + "_link",
			PasswordHash: hash,
			IsActive:     true,
			Roles:        []string{role},
		}
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Maybe()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	}

	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth)

	svc := NewLinkService(linkRepo, incRepo, outRepo, auth, journalSvc)
	return svc, linkRepo, incRepo, outRepo, auth
}

func TestLinkService_LinkDocuments(t *testing.T) {
	// Регистрация связи между двумя документами (например, ответ на письмо)
	sourceID := uuid.New()
	targetID := uuid.New()

	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _, auth := setupLinkService(t, "clerk")
		userIDStr := auth.GetCurrentUserID()
		userID, _ := uuid.Parse(userIDStr)

		repo.On("Create", context.Background(), mock.MatchedBy(func(link *models.DocumentLink) bool {
			return link.SourceID == sourceID && link.TargetID == targetID && link.LinkType == "ответ" && link.CreatedBy == userID
		})).Return(nil).Once()

		result, err := svc.LinkDocuments(sourceID.String(), targetID.String(), "incoming", "outgoing", "ответ")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ответ", result.LinkType)
		assert.Equal(t, sourceID.String(), result.SourceID)
	})

	t.Run("запрещено связывать с собой", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "clerk")
		result, err := svc.LinkDocuments(sourceID.String(), sourceID.String(), "incoming", "incoming", "копия")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot link document to itself")
		assert.Nil(t, result)
	})

	t.Run("невалидный source ID", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "clerk")
		result, err := svc.LinkDocuments("invalid", targetID.String(), "incoming", "outgoing", "ответ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid source ID")
		assert.Nil(t, result)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "")
		result, err := svc.LinkDocuments(sourceID.String(), targetID.String(), "incoming", "outgoing", "ответ")
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrUnauthorized)
		assert.Nil(t, result)
	})

	t.Run("executor не может создавать связи", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "executor")
		result, err := svc.LinkDocuments(sourceID.String(), targetID.String(), "incoming", "outgoing", "ответ")
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

func TestLinkService_UnlinkDocument(t *testing.T) {
	// Разрыв (удаление) связи между документами
	linkID := uuid.New()

	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _, _ := setupLinkService(t, "clerk")
		rootID := uuid.New()
		targetID := uuid.New()

		repo.On("GetByID", context.Background(), linkID).Return(&models.DocumentLink{
			ID:         linkID,
			SourceID:   rootID,
			SourceType: "incoming",
			TargetID:   targetID,
			TargetType: "outgoing",
		}, nil).Once()
		repo.On("Delete", context.Background(), linkID).Return(nil).Once()

		err := svc.UnlinkDocument(linkID.String())
		require.NoError(t, err)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "clerk")
		err := svc.UnlinkDocument("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID")
	})

	t.Run("executor не может удалять связи", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "executor")
		err := svc.UnlinkDocument(linkID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
	})
}

func TestLinkService_GetDocumentLinks(t *testing.T) {
	// Получение всех прямых связей для конкретного документа
	docID := uuid.New()

	t.Run("успех", func(t *testing.T) {
		svc, repo, _, _, _ := setupLinkService(t, "clerk")
		mockValues := []models.DocumentLink{
			{ID: uuid.New(), SourceID: docID, TargetID: uuid.New(), LinkType: "ответ"},
		}
		repo.On("GetByDocumentID", context.Background(), docID).Return(mockValues, nil).Once()

		result, err := svc.GetDocumentLinks(docID.String())
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "ответ", result[0].LinkType)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "clerk")
		result, err := svc.GetDocumentLinks("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document ID")
		assert.Nil(t, result)
	})

	t.Run("executor не может читать связи", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "executor")
		result, err := svc.GetDocumentLinks(docID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

func TestLinkService_GetDocumentFlow(t *testing.T) {
	// Получение полного графа (цепочки) связанных документов для визуализации истории
	rootID := uuid.New()
	targetID := uuid.New()

	t.Run("успех - граф со связями", func(t *testing.T) {
		svc, repo, incRepo, outRepo, _ := setupLinkService(t, "clerk")

		mockLinks := []models.DocumentLink{
			{ID: uuid.New(), SourceID: rootID, SourceType: "incoming", TargetID: targetID, TargetType: "outgoing", LinkType: "ответ"},
		}
		repo.On("GetGraph", context.Background(), rootID).Return(mockLinks, nil).Once()

		incDoc := &models.IncomingDocument{ID: rootID, IncomingNumber: "ВХ-1", Content: "Тест вх", IncomingDate: time.Now()}
		outDoc := &models.OutgoingDocument{ID: targetID, OutgoingNumber: "ИСХ-2", Content: "Тест исх", OutgoingDate: time.Now()}

		incRepo.On("GetByID", rootID).Return(incDoc, nil).Once()
		outRepo.On("GetByID", targetID).Return(outDoc, nil).Once()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Nodes, 2)
		assert.Len(t, result.Edges, 1)
	})

	t.Run("пустой граф", func(t *testing.T) {
		svc, repo, _, _, _ := setupLinkService(t, "clerk")
		repo.On("GetGraph", context.Background(), rootID).Return([]models.DocumentLink{}, nil).Once()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Nodes)
		assert.Empty(t, result.Edges)
	})

	t.Run("ошибка базы", func(t *testing.T) {
		svc, repo, _, _, _ := setupLinkService(t, "clerk")
		repo.On("GetGraph", context.Background(), rootID).Return(nil, errors.New("db error")).Once()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
		assert.Nil(t, result)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "clerk")
		result, err := svc.GetDocumentFlow("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document ID")
		assert.Nil(t, result)
	})

	t.Run("admin не может читать граф связей", func(t *testing.T) {
		svc, _, _, _, auth := setupLinkService(t, "admin")
		require.Equal(t, "admin", auth.GetActiveRole())

		result, err := svc.GetDocumentFlow(rootID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

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
	return setupLinkServiceWithAccessStore(t, role, newRoleMappedDocumentAccessStore(role))
}

func setupLinkServiceWithAccessStore(t *testing.T, role string, accessStore DocumentAccessStore) (*LinkService, *mocks.LinkStore, *mocks.IncomingDocStore, *mocks.OutgoingDocStore, *AuthService) {
	t.Helper()
	linkRepo := mocks.NewLinkStore(t)
	incRepo := mocks.NewIncomingDocStore(t)
	outRepo := mocks.NewOutgoingDocStore(t)
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
			Login:        role + "_link",
			PasswordHash: hash,
			IsActive:     true,
		}
		userRepo.On("GetByLogin", user.Login).Return(user, nil).Maybe()
		_, err := auth.Login(user.Login, password)
		require.NoError(t, err)
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	}
	assignmentRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	ackRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()

	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	accessSvc := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, accessStore, nil, incRepo, outRepo)
	journalSvc := NewJournalService(journalRepo, auth, accessSvc)

	svc := NewLinkService(linkRepo, incRepo, outRepo, accessSvc, auth, journalSvc)
	return svc, linkRepo, incRepo, outRepo, auth
}

type kindActionDocumentAccessStore struct {
	allowed map[models.DocumentKind]map[string]bool
}

func (s *kindActionDocumentAccessStore) HasPermission(kindCode, action string, departmentID, userID string) (bool, error) {
	actions := s.allowed[models.NormalizeDocumentKind(kindCode)]
	return actions[action], nil
}

func (s *kindActionDocumentAccessStore) HasSystemPermission(permission, userID string) (bool, error) {
	return false, nil
}

func (s *kindActionDocumentAccessStore) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	return &models.UserDocumentAccessProfile{}, nil
}

func (s *kindActionDocumentAccessStore) ReplaceUserAccessProfile(userID string, systemPermissions []models.UserSystemPermissionRule, permissions []models.UserDocumentPermissionRule) error {
	return nil
}

func TestLinkService_LinkDocuments(t *testing.T) {
	// Регистрация связи между двумя документами (например, ответ на письмо)
	sourceID := uuid.New()
	targetID := uuid.New()

	t.Run("успех", func(t *testing.T) {
		svc, repo, incRepo, outRepo, auth := setupLinkService(t, "clerk")
		userIDStr := auth.GetCurrentUserID()
		userID, _ := uuid.Parse(userIDStr)
		incRepo.On("GetByID", sourceID).Return(&models.IncomingDocument{ID: sourceID}, nil).Once()
		incRepo.On("GetByID", targetID).Return((*models.IncomingDocument)(nil), nil).Once()
		outRepo.On("GetByID", targetID).Return(&models.OutgoingDocument{ID: targetID}, nil).Once()

		repo.On("Create", context.Background(), mock.MatchedBy(func(link *models.DocumentLink) bool {
			return link.SourceID == sourceID && link.TargetID == targetID && link.LinkType == "ответ" && link.CreatedBy == userID
		})).Return(nil).Once()

		result, err := svc.LinkDocuments(sourceID.String(), targetID.String(), "ответ")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ответ", result.LinkType)
		assert.Equal(t, sourceID.String(), result.SourceID)
	})

	t.Run("запрещено связывать с собой", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "clerk")
		result, err := svc.LinkDocuments(sourceID.String(), sourceID.String(), "копия")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot link document to itself")
		assert.Nil(t, result)
	})

	t.Run("невалидный source ID", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "clerk")
		result, err := svc.LinkDocuments("invalid", targetID.String(), "ответ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid source ID")
		assert.Nil(t, result)
	})

	t.Run("не авторизован", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "")
		result, err := svc.LinkDocuments(sourceID.String(), targetID.String(), "ответ")
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrUnauthorized)
		assert.Nil(t, result)
	})

	t.Run("executor не может создавать связи", func(t *testing.T) {
		svc, _, incRepo, outRepo, _ := setupLinkService(t, "executor")
		incRepo.On("GetByID", sourceID).Return(&models.IncomingDocument{ID: sourceID}, nil).Once()
		incRepo.On("GetByID", targetID).Return((*models.IncomingDocument)(nil), nil).Once()
		outRepo.On("GetByID", targetID).Return(&models.OutgoingDocument{ID: targetID}, nil).Once()
		result, err := svc.LinkDocuments(sourceID.String(), targetID.String(), "ответ")
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

func TestLinkService_UnlinkDocument(t *testing.T) {
	// Разрыв (удаление) связи между документами
	linkID := uuid.New()

	t.Run("успех", func(t *testing.T) {
		svc, repo, incRepo, outRepo, _ := setupLinkService(t, "clerk")
		rootID := uuid.New()
		targetID := uuid.New()
		incRepo.On("GetByID", rootID).Return(&models.IncomingDocument{ID: rootID}, nil).Once()
		incRepo.On("GetByID", targetID).Return((*models.IncomingDocument)(nil), nil).Once()
		outRepo.On("GetByID", targetID).Return(&models.OutgoingDocument{ID: targetID}, nil).Once()

		repo.On("GetByID", context.Background(), linkID).Return(&models.DocumentLink{
			ID:         linkID,
			SourceID:   rootID,
			SourceKind: models.DocumentKindIncomingLetter,
			TargetID:   targetID,
			TargetKind: models.DocumentKindOutgoingLetter,
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
		svc, repo, incRepo, outRepo, _ := setupLinkService(t, "executor")
		sourceID := uuid.New()
		targetID := uuid.New()
		repo.On("GetByID", context.Background(), linkID).Return(&models.DocumentLink{
			ID:         linkID,
			SourceID:   sourceID,
			SourceKind: models.DocumentKindIncomingLetter,
			TargetID:   targetID,
			TargetKind: models.DocumentKindOutgoingLetter,
		}, nil).Once()
		incRepo.On("GetByID", sourceID).Return(&models.IncomingDocument{ID: sourceID}, nil).Once()
		incRepo.On("GetByID", targetID).Return((*models.IncomingDocument)(nil), nil).Once()
		outRepo.On("GetByID", targetID).Return(&models.OutgoingDocument{ID: targetID}, nil).Once()
		err := svc.UnlinkDocument(linkID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
	})
}

func TestLinkService_GetDocumentLinks(t *testing.T) {
	// Получение всех прямых связей для конкретного документа
	docID := uuid.New()

	t.Run("успех", func(t *testing.T) {
		svc, repo, incRepo, _, _ := setupLinkService(t, "clerk")
		targetID := uuid.New()
		incRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID}, nil).Maybe()
		incRepo.On("GetByID", targetID).Return(&models.IncomingDocument{ID: targetID}, nil).Maybe()
		mockValues := []models.DocumentLink{
			{ID: uuid.New(), SourceID: docID, TargetID: targetID, LinkType: "ответ"},
		}
		repo.On("GetByDocumentID", context.Background(), docID).Return(mockValues, nil).Once()

		result, err := svc.GetDocumentLinks(docID.String())
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "ответ", result[0].LinkType)
	})

	t.Run("скрывает связи с недоступным документом", func(t *testing.T) {
		accessStore := &kindActionDocumentAccessStore{
			allowed: map[models.DocumentKind]map[string]bool{
				models.DocumentKindIncomingLetter: map[string]bool{
					"link": true,
					"read": true,
				},
			},
		}
		svc, repo, incRepo, outRepo, _ := setupLinkServiceWithAccessStore(t, "clerk", accessStore)
		targetID := uuid.New()
		incRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID}, nil).Maybe()
		incRepo.On("GetByID", targetID).Return((*models.IncomingDocument)(nil), nil).Maybe()
		outRepo.On("GetByID", targetID).Return(&models.OutgoingDocument{ID: targetID}, nil).Maybe()
		mockValues := []models.DocumentLink{
			{
				ID:         uuid.New(),
				SourceID:   docID,
				SourceKind: models.DocumentKindIncomingLetter,
				TargetID:   targetID,
				TargetKind: models.DocumentKindOutgoingLetter,
				LinkType:   "ответ",
			},
		}
		repo.On("GetByDocumentID", context.Background(), docID).Return(mockValues, nil).Once()

		result, err := svc.GetDocumentLinks(docID.String())
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("невалидный ID", func(t *testing.T) {
		svc, _, _, _, _ := setupLinkService(t, "clerk")
		result, err := svc.GetDocumentLinks("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document ID")
		assert.Nil(t, result)
	})

	t.Run("executor не может читать связи", func(t *testing.T) {
		svc, _, incRepo, _, _ := setupLinkService(t, "executor")
		incRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID}, nil).Once()
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
			{ID: uuid.New(), SourceID: rootID, SourceKind: models.DocumentKindIncomingLetter, TargetID: targetID, TargetKind: models.DocumentKindOutgoingLetter, LinkType: "ответ"},
		}
		repo.On("GetGraph", context.Background(), rootID).Return(mockLinks, nil).Once()

		incDoc := &models.IncomingDocument{ID: rootID, IncomingNumber: "ВХ-1", Content: "Тест вх", IncomingDate: time.Now()}
		outDoc := &models.OutgoingDocument{ID: targetID, OutgoingNumber: "ИСХ-2", Content: "Тест исх", OutgoingDate: time.Now()}

		incRepo.On("GetByID", rootID).Return(incDoc, nil).Maybe()
		incRepo.On("GetByID", targetID).Return((*models.IncomingDocument)(nil), nil).Maybe()
		outRepo.On("GetByID", targetID).Return(outDoc, nil).Maybe()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Nodes, 2)
		assert.Len(t, result.Edges, 1)
	})

	t.Run("скрывает доступные узлы за недоступным мостом", func(t *testing.T) {
		accessStore := &kindActionDocumentAccessStore{
			allowed: map[models.DocumentKind]map[string]bool{
				models.DocumentKindIncomingLetter: map[string]bool{
					"link": true,
					"read": true,
				},
			},
		}
		svc, repo, incRepo, outRepo, _ := setupLinkServiceWithAccessStore(t, "clerk", accessStore)
		blockedID := uuid.New()
		visibleID := uuid.New()
		visibleChildID := uuid.New()

		mockLinks := []models.DocumentLink{
			{ID: uuid.New(), SourceID: rootID, SourceKind: models.DocumentKindIncomingLetter, TargetID: blockedID, TargetKind: models.DocumentKindOutgoingLetter, LinkType: "ответ"},
			{ID: uuid.New(), SourceID: blockedID, SourceKind: models.DocumentKindOutgoingLetter, TargetID: visibleID, TargetKind: models.DocumentKindIncomingLetter, LinkType: "связано"},
			{ID: uuid.New(), SourceID: visibleID, SourceKind: models.DocumentKindIncomingLetter, TargetID: visibleChildID, TargetKind: models.DocumentKindIncomingLetter, LinkType: "связано"},
		}
		repo.On("GetGraph", context.Background(), rootID).Return(mockLinks, nil).Once()

		incRepo.On("GetByID", rootID).Return(&models.IncomingDocument{ID: rootID}, nil).Maybe()
		incRepo.On("GetByID", blockedID).Return((*models.IncomingDocument)(nil), nil).Maybe()
		outRepo.On("GetByID", blockedID).Return(&models.OutgoingDocument{ID: blockedID}, nil).Maybe()
		incRepo.On("GetByID", visibleID).Return(&models.IncomingDocument{ID: visibleID}, nil).Maybe()
		incRepo.On("GetByID", visibleChildID).Return(&models.IncomingDocument{ID: visibleChildID}, nil).Maybe()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Nodes)
		assert.Empty(t, result.Edges)
	})

	t.Run("пустой граф", func(t *testing.T) {
		svc, repo, incRepo, _, _ := setupLinkService(t, "clerk")
		incRepo.On("GetByID", rootID).Return(&models.IncomingDocument{ID: rootID}, nil).Once()
		repo.On("GetGraph", context.Background(), rootID).Return([]models.DocumentLink{}, nil).Once()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Nodes)
		assert.Empty(t, result.Edges)
	})

	t.Run("ошибка базы", func(t *testing.T) {
		svc, repo, incRepo, _, _ := setupLinkService(t, "clerk")
		incRepo.On("GetByID", rootID).Return(&models.IncomingDocument{ID: rootID}, nil).Once()
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
		svc, _, incRepo, _, _ := setupLinkService(t, "admin")
		incRepo.On("GetByID", rootID).Return(&models.IncomingDocument{ID: rootID}, nil).Once()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

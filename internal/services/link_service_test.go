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

	svc := NewLinkService(linkRepo, incRepo, outRepo, nil, nil, accessSvc, auth, journalSvc)
	return svc, linkRepo, incRepo, outRepo, auth
}

type kindActionDocumentAccessStore struct {
	allowed map[models.DocumentKind]map[string]bool
}

type mapDocumentStore struct {
	docs map[uuid.UUID]*models.Document
}

func (s *mapDocumentStore) GetByID(id uuid.UUID) (*models.Document, error) {
	return s.docs[id], nil
}

type mapCitizenAppealDocStore struct {
	docs map[uuid.UUID]*models.CitizenAppealDocument
}

type mapAdministrativeOrderDocStore struct {
	docs map[uuid.UUID]*models.AdministrativeOrderDocument
}

func (s *mapCitizenAppealDocStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.CitizenAppealDocument], error) {
	return nil, nil
}

func (s *mapCitizenAppealDocStore) GetByID(id uuid.UUID) (*models.CitizenAppealDocument, error) {
	return s.docs[id], nil
}

func (s *mapCitizenAppealDocStore) Create(req models.CreateCitizenAppealDocRequest) (*models.CitizenAppealDocument, error) {
	return nil, nil
}

func (s *mapCitizenAppealDocStore) Update(req models.UpdateCitizenAppealDocRequest) (*models.CitizenAppealDocument, error) {
	return nil, nil
}

func (s *mapCitizenAppealDocStore) GetCount() (int, error) {
	return 0, nil
}

func (s *mapAdministrativeOrderDocStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.AdministrativeOrderDocument], error) {
	return nil, nil
}

func (s *mapAdministrativeOrderDocStore) GetByID(id uuid.UUID) (*models.AdministrativeOrderDocument, error) {
	return s.docs[id], nil
}

func (s *mapAdministrativeOrderDocStore) Create(req models.CreateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *mapAdministrativeOrderDocStore) Update(req models.UpdateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *mapAdministrativeOrderDocStore) GetAcknowledgmentPersonByID(id uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *mapAdministrativeOrderDocStore) GetAcknowledgmentPeople(documentID uuid.UUID) ([]models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *mapAdministrativeOrderDocStore) MarkAcknowledgmentPerson(id uuid.UUID, acknowledgedBy uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *mapAdministrativeOrderDocStore) CancelByLink(id uuid.UUID, cancelledAt time.Time) error {
	return nil
}

func (s *mapAdministrativeOrderDocStore) GetCount() (int, error) {
	return 0, nil
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

func TestValidateDocumentLinkType(t *testing.T) {
	tests := []struct {
		name       string
		sourceKind models.DocumentKind
		targetKind models.DocumentKind
		linkType   string
		wantErr    bool
	}{
		{
			name:       "order amends between administrative orders",
			sourceKind: models.DocumentKindAdministrativeOrder,
			targetKind: models.DocumentKindAdministrativeOrder,
			linkType:   "order_amends",
		},
		{
			name:       "order cancels between administrative orders",
			sourceKind: models.DocumentKindAdministrativeOrder,
			targetKind: models.DocumentKindAdministrativeOrder,
			linkType:   "order_cancels",
		},
		{
			name:       "order link rejects non order source",
			sourceKind: models.DocumentKindIncomingLetter,
			targetKind: models.DocumentKindAdministrativeOrder,
			linkType:   "order_amends",
			wantErr:    true,
		},
		{
			name:       "order link rejects non order target",
			sourceKind: models.DocumentKindAdministrativeOrder,
			targetKind: models.DocumentKindOutgoingLetter,
			linkType:   "order_cancels",
			wantErr:    true,
		},
		{
			name:       "custom link type is accepted for mixed kinds",
			sourceKind: models.DocumentKindIncomingLetter,
			targetKind: models.DocumentKindOutgoingLetter,
			linkType:   "ответ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDocumentLinkType(tt.sourceKind, tt.targetKind, tt.linkType)
			if tt.wantErr {
				require.Error(t, err)
				appErr, ok := models.AsAppError(err)
				require.True(t, ok)
				assert.Equal(t, "VALIDATION_ERROR", appErr.Kind)
				return
			}
			require.NoError(t, err)
		})
	}
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

	t.Run("обращение отображает ФИО заявителя как отправителя", func(t *testing.T) {
		svc, repo, incRepo, _, _ := setupLinkService(t, "clerk")
		appealID := uuid.New()
		registrationDate := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)

		svc.access.documentRepo = &mapDocumentStore{
			docs: map[uuid.UUID]*models.Document{
				rootID: {
					ID:                 rootID,
					Kind:               models.DocumentKindIncomingLetter,
					RegistrationNumber: "ВХ-1",
					RegistrationDate:   registrationDate,
				},
				appealID: {
					ID:                 appealID,
					Kind:               models.DocumentKindCitizenAppeal,
					RegistrationNumber: "ОБ-7",
					RegistrationDate:   registrationDate,
					Content:            "Просьба заявителя",
				},
			},
		}
		svc.citizenAppealDocRepo = &mapCitizenAppealDocStore{
			docs: map[uuid.UUID]*models.CitizenAppealDocument{
				appealID: {
					ID:                 appealID,
					RegistrationNumber: "ОБ-7",
					RegistrationDate:   registrationDate,
					Content:            "Просьба заявителя",
					ApplicantFullName:  "Иванов Иван Иванович",
				},
			},
		}

		mockLinks := []models.DocumentLink{
			{ID: uuid.New(), SourceID: rootID, SourceKind: models.DocumentKindIncomingLetter, TargetID: appealID, TargetKind: models.DocumentKindCitizenAppeal, LinkType: "связано"},
		}
		repo.On("GetGraph", context.Background(), rootID).Return(mockLinks, nil).Once()
		incRepo.On("GetByID", rootID).Return(&models.IncomingDocument{ID: rootID, IncomingNumber: "ВХ-1", IncomingDate: registrationDate}, nil).Maybe()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.NoError(t, err)
		require.NotNil(t, result)

		var appealNode *models.GraphNode
		for i := range result.Nodes {
			if result.Nodes[i].ID == appealID.String() {
				appealNode = &result.Nodes[i]
				break
			}
		}
		require.NotNil(t, appealNode)
		assert.Equal(t, string(models.DocumentKindCitizenAppeal), appealNode.KindCode)
		assert.Equal(t, "Иванов Иван Иванович", appealNode.Sender)
	})

	t.Run("приказ передает статус активности в узел графа", func(t *testing.T) {
		svc, repo, _, _, _ := setupLinkService(t, "clerk")
		cancelledOrderID := uuid.New()
		registrationDate := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)

		svc.access.documentRepo = &mapDocumentStore{
			docs: map[uuid.UUID]*models.Document{
				rootID: {
					ID:                 rootID,
					Kind:               models.DocumentKindAdministrativeOrder,
					RegistrationNumber: "П-1",
					RegistrationDate:   registrationDate,
				},
				cancelledOrderID: {
					ID:                 cancelledOrderID,
					Kind:               models.DocumentKindAdministrativeOrder,
					RegistrationNumber: "П-2",
					RegistrationDate:   registrationDate,
				},
			},
		}
		svc.administrativeOrderRepo = &mapAdministrativeOrderDocStore{
			docs: map[uuid.UUID]*models.AdministrativeOrderDocument{
				rootID: {
					ID:          rootID,
					OrderNumber: "П-1",
					OrderDate:   registrationDate,
					Title:       "Действующий приказ",
					IsActive:    true,
				},
				cancelledOrderID: {
					ID:          cancelledOrderID,
					OrderNumber: "П-2",
					OrderDate:   registrationDate,
					Title:       "Недействующий приказ",
					IsActive:    false,
				},
			},
		}

		mockLinks := []models.DocumentLink{
			{ID: uuid.New(), SourceID: rootID, SourceKind: models.DocumentKindAdministrativeOrder, TargetID: cancelledOrderID, TargetKind: models.DocumentKindAdministrativeOrder, LinkType: "order_cancels"},
		}
		repo.On("GetGraph", context.Background(), rootID).Return(mockLinks, nil).Once()

		result, err := svc.GetDocumentFlow(rootID.String())
		require.NoError(t, err)
		require.NotNil(t, result)

		var cancelledOrderNode *models.GraphNode
		for i := range result.Nodes {
			if result.Nodes[i].ID == cancelledOrderID.String() {
				cancelledOrderNode = &result.Nodes[i]
				break
			}
		}
		require.NotNil(t, cancelledOrderNode)
		require.NotNil(t, cancelledOrderNode.IsActive)
		assert.False(t, *cancelledOrderNode.IsActive)
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

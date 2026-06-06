package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"

	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func setupAssignmentService(t *testing.T, role string) (
	*AssignmentService, *mocks.AssignmentStore, *mocks.UserStore, *AuthService, *mocks.IncomingDocStore,
) {
	t.Helper()
	assignmentRepo := mocks.NewAssignmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:                    uuid.New(),
		Login:                 role + "_user",
		PasswordHash:          hash,
		IsDocumentParticipant: role != "" && role != "admin",
		IsActive:              true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth, nil)
	incomingRepo := mocks.NewIncomingDocStore(t)
	outgoingRepo := mocks.NewOutgoingDocStore(t)
	incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
		return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
		return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	assignmentRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	accessSvc := NewDocumentAccessService(auth, nil, assignmentRepo, nil, newRoleMappedDocumentAccessStore(role), nil, incomingRepo, outgoingRepo)

	svc := NewAssignmentService(assignmentRepo, userRepo, auth, journalSvc, accessSvc)
	return svc, assignmentRepo, userRepo, auth, incomingRepo
}

func setupAssignmentServiceNotAuth(t *testing.T) (*AssignmentService, *mocks.AssignmentStore) {
	t.Helper()
	assignmentRepo := mocks.NewAssignmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth, nil)
	incomingRepo := mocks.NewIncomingDocStore(t)
	outgoingRepo := mocks.NewOutgoingDocStore(t)
	incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
		return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
		return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	assignmentRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	accessSvc := NewDocumentAccessService(auth, nil, assignmentRepo, nil, newRoleMappedDocumentAccessStore(), nil, incomingRepo, outgoingRepo)

	svc := NewAssignmentService(assignmentRepo, userRepo, auth, journalSvc, accessSvc)
	return svc, assignmentRepo
}

type kindActionAccessStore struct {
	allowed map[string]map[string]bool
}

func newKindActionAccessStore(allowed map[string][]string) DocumentAccessStore {
	store := &kindActionAccessStore{allowed: make(map[string]map[string]bool, len(allowed))}
	for kindCode, actions := range allowed {
		store.allowed[kindCode] = make(map[string]bool, len(actions))
		for _, action := range actions {
			store.allowed[kindCode][action] = true
		}
	}
	return store
}

func (s *kindActionAccessStore) HasPermission(kindCode, action string, departmentID, userID string) (bool, error) {
	return s.allowed[kindCode][action], nil
}

func (s *kindActionAccessStore) HasSystemPermission(permission, userID string) (bool, error) {
	return false, nil
}

func (s *kindActionAccessStore) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	return &models.UserDocumentAccessProfile{}, nil
}

func (s *kindActionAccessStore) ReplaceUserAccessProfile(userID string, systemPermissions []models.UserSystemPermissionRule, permissions []models.UserDocumentPermissionRule) error {
	return nil
}

// ---------- TestAssignmentService_Create ----------

func TestAssignmentService_Create(t *testing.T) {
	// Создание нового поручения по документу
	docID := uuid.New()
	execID := uuid.New()

	t.Run("success with deadline", func(t *testing.T) {
		svc, repo, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		expected := &models.Assignment{
			ID:         uuid.New(),
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Выполнить",
			Status:     "new",
		}
		repo.On("Create", docID, execID, "Выполнить",
			mock.AnythingOfType("*time.Time"), []string(nil),
		).Return(expected, nil).Once()

		result, err := svc.Create(docID.String(), execID.String(), "Выполнить", "2025-12-31", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expected.ID.String(), result.ID)
	})

	t.Run("success without deadline", func(t *testing.T) {
		svc, repo, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		expected := &models.Assignment{
			ID:         uuid.New(),
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Выполнить",
			Status:     "new",
		}
		repo.On("Create", docID, execID, "Выполнить",
			(*time.Time)(nil), []string(nil),
		).Return(expected, nil).Once()

		result, err := svc.Create(docID.String(), execID.String(), "Выполнить", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc, _ := setupAssignmentServiceNotAuth(t)

		result, err := svc.Create(docID.String(), execID.String(), "Выполнить", "", nil)
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})

	t.Run("invalid document ID", func(t *testing.T) {
		svc, _, _, _, _ := setupAssignmentService(t, "clerk")

		result, err := svc.Create("not-a-uuid", execID.String(), "Выполнить", "", nil)
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID документа")
		assert.Nil(t, result)
	})

	t.Run("invalid executor ID", func(t *testing.T) {
		svc, _, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		result, err := svc.Create(docID.String(), "not-a-uuid", "Выполнить", "", nil)
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID исполнителя")
		assert.Nil(t, result)
	})
}

func TestAssignmentService_CreateEmitsUserEvents(t *testing.T) {
	docID := uuid.New()
	execID := uuid.New()
	coExecID := uuid.New()

	svc, repo, _, auth, incomingRepo := setupAssignmentService(t, "clerk")
	eventStore := &fakeUserEventStore{}
	svc.events = NewUserEventService(eventStore, auth)
	incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{
		ID:             docID,
		NomenclatureID: uuid.New(),
		IncomingNumber: "ВХ-1",
	}, nil).Maybe()

	expected := &models.Assignment{
		ID:             uuid.New(),
		DocumentID:     docID,
		DocumentKind:   "incoming_letter",
		DocumentNumber: "ВХ-1",
		ExecutorID:     execID,
		Content:        "Выполнить",
		Status:         "new",
		CoExecutorIDs:  []string{coExecID.String()},
	}
	repo.On("Create", docID, execID, "Выполнить",
		(*time.Time)(nil), []string{coExecID.String()},
	).Return(expected, nil).Once()

	result, err := svc.Create(docID.String(), execID.String(), "Выполнить", "", []string{coExecID.String()})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, eventStore.created, 2)
	assert.Equal(t, execID, eventStore.created[0].RecipientUserID)
	assert.Equal(t, coExecID, eventStore.created[1].RecipientUserID)
	assert.Equal(t, models.UserEventAssignmentCreated, eventStore.created[0].EventType)
	assert.Equal(t, "Новое поручение", eventStore.created[0].Title)
	assert.Equal(t, "ВХ-1", eventStore.created[0].DocumentNumber)
}

// ---------- TestAssignmentService_Update ----------

func TestAssignmentService_Update(t *testing.T) {
	// Обновление параметров существующего поручения (текст, исполнитель, дедлайн)
	assignmentID := uuid.New()
	execID := uuid.New()
	docID := uuid.New()

	existing := &models.Assignment{
		ID:         assignmentID,
		DocumentID: docID,
		ExecutorID: execID,
		Content:    "Старое",
		Status:     "new",
	}

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Update", assignmentID, execID, "Новое",
			(*time.Time)(nil), "new", "", (*time.Time)(nil), []string(nil),
		).Return(&models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Новое",
			Status:     "new",
		}, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Новое", result.Content)
	})

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Update", assignmentID, execID, "Обновлено",
			(*time.Time)(nil), "new", "", (*time.Time)(nil), []string(nil),
		).Return(&models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Обновлено",
			Status:     "new",
		}, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Обновлено", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "executor")
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(nil, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.Error(t, err)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", appErr.Kind)
		assert.Equal(t, 404, appErr.Code)
		assert.Nil(t, result)
	})

	t.Run("finished not admin", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		finished := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "finished",
		}

		repo.On("GetByID", assignmentID).Return(finished, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.Error(t, err)
		requireAppError(t, err, "CONFLICT", 409, "завершённое")
		assert.Nil(t, result)
	})
}

// ---------- TestAssignmentService_UpdateStatus ----------

func TestAssignmentService_UpdateStatus(t *testing.T) {
	// Изменение статуса поручения (в работу, на проверку, завершено и т.д.)
	assignmentID := uuid.New()
	execID := uuid.New()
	docID := uuid.New()

	existing := &models.Assignment{
		ID:         assignmentID,
		DocumentID: docID,
		ExecutorID: execID,
		Content:    "Контент",
		Status:     "new",
	}

	t.Run("executor to in_progress", func(t *testing.T) {
		_, repo, userRepo, _, _ := setupAssignmentService(t, "executor")
		// Override auth currentUser ID to match executor
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		executorUser := &models.User{
			ID:                    execID,
			Login:                 "exec",
			PasswordHash:          hash,
			IsDocumentParticipant: true,
			IsActive:              true,
		}
		authSvc := NewAuthService(nil, userRepo)
		userRepo.On("GetByLogin", "exec").Return(executorUser, nil).Once()
		authSvc.Login("exec", password)
		userRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()
		journalRepo := mocks.NewJournalStore(t)
		journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
		journalSvc := NewJournalService(journalRepo, authSvc, nil)

		incomingRepo := mocks.NewIncomingDocStore(t)
		outgoingRepo := mocks.NewOutgoingDocStore(t)
		incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
			return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
			return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		repo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
		accessSvc := NewDocumentAccessService(authSvc, nil, repo, nil, newRoleMappedDocumentAccessStore("executor"), nil, incomingRepo, outgoingRepo)
		svc2 := NewAssignmentService(repo, userRepo, authSvc, journalSvc, accessSvc)

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Update", assignmentID, execID, "Контент",
			(*time.Time)(nil), "in_progress", "",
			(*time.Time)(nil), []string(nil),
		).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "in_progress",
		}, nil).Once()

		result, err := svc2.UpdateStatus(assignmentID.String(), "in_progress", "")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "in_progress", result.Status)
	})

	t.Run("executor to completed", func(t *testing.T) {
		_, repo, userRepo, _, _ := setupAssignmentService(t, "executor")
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		executorUser := &models.User{
			ID:                    execID,
			Login:                 "exec2",
			PasswordHash:          hash,
			IsDocumentParticipant: true,
			IsActive:              true,
		}
		authSvc := NewAuthService(nil, userRepo)
		userRepo.On("GetByLogin", "exec2").Return(executorUser, nil).Once()
		authSvc.Login("exec2", password)
		userRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()
		journalRepo := mocks.NewJournalStore(t)
		journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
		journalSvc := NewJournalService(journalRepo, authSvc, nil)

		incomingRepo := mocks.NewIncomingDocStore(t)
		outgoingRepo := mocks.NewOutgoingDocStore(t)
		incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
			return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
			return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		repo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
		accessSvc := NewDocumentAccessService(authSvc, nil, repo, nil, newRoleMappedDocumentAccessStore("executor"), nil, incomingRepo, outgoingRepo)
		svc2 := NewAssignmentService(repo, userRepo, authSvc, journalSvc, accessSvc)

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Update", assignmentID, execID, "Контент",
			(*time.Time)(nil), "completed", "Отчёт",
			mock.AnythingOfType("*time.Time"), []string(nil),
		).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "completed",
			Report: "Отчёт",
		}, nil).Once()

		result, err := svc2.UpdateStatus(assignmentID.String(), "completed", "Отчёт")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "completed", result.Status)
	})

	t.Run("clerk to finished from completed", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		completedAt := time.Now()
		completedAssignment := &models.Assignment{
			ID:          assignmentID,
			DocumentID:  docID,
			ExecutorID:  execID,
			Content:     "Контент",
			Status:      "completed",
			Report:      "Отчёт исполнителя",
			CompletedAt: &completedAt,
		}

		repo.On("GetByID", assignmentID).Return(completedAssignment, nil).Once()
		repo.On("Update", assignmentID, execID, "Контент",
			(*time.Time)(nil), "finished", "Отчёт исполнителя",
			completedAssignment.CompletedAt, []string(nil),
		).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "finished",
			Report: "Отчёт исполнителя",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "finished", result.Status)
	})

	t.Run("clerk to returned requires reason", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		completedAssignment := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "completed",
			Report:     "Отчёт исполнителя",
		}

		repo.On("GetByID", assignmentID).Return(completedAssignment, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "returned", "")
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "причина возврата обязательна")
		assert.Nil(t, result)
	})

	t.Run("clerk to returned with reason", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		completedAssignment := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "completed",
			Report:     "Отчёт исполнителя",
		}

		repo.On("GetByID", assignmentID).Return(completedAssignment, nil).Once()
		repo.On("Update", assignmentID, execID, "Контент",
			(*time.Time)(nil), "returned", "Нужно исправить замечания",
			(*time.Time)(nil), []string(nil),
		).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "returned",
			Report: "Нужно исправить замечания",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "returned", "Нужно исправить замечания")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "returned", result.Status)
		assert.Equal(t, "Нужно исправить замечания", result.Report)
	})

	t.Run("admin forbidden", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "admin")
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "недостаточно прав")
		assert.Nil(t, result)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "executor")
		// Executor with different ID than assignment executor — can't set "finished"
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAssignmentService_UpdateStatusEmitsUserEvents(t *testing.T) {
	assignmentID := uuid.New()
	docID := uuid.New()
	execID := uuid.New()

	t.Run("completed notifies controllers", func(t *testing.T) {
		svc, repo, userRepo, auth, _ := setupAssignmentService(t, "executor")
		executorID, _ := uuid.Parse(auth.GetCurrentUserID())
		svc.access.accessRepo = newKindActionAccessStore(map[string][]string{
			"incoming_letter": {"assign"},
		})
		eventStore := &fakeUserEventStore{}
		svc.events = NewUserEventService(eventStore, auth)
		controllerID := uuid.New()
		userRepo.On("GetAll").Return([]models.User{
			{ID: controllerID, IsActive: true},
			{ID: executorID, IsActive: true},
		}, nil).Once()

		existing := &models.Assignment{
			ID:           assignmentID,
			DocumentID:   docID,
			DocumentKind: "incoming_letter",
			ExecutorID:   executorID,
			Content:      "Контент",
			Status:       "in_progress",
		}
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Update", assignmentID, executorID, "Контент",
			(*time.Time)(nil), "completed", "Отчет",
			mock.AnythingOfType("*time.Time"), []string(nil),
		).Return(&models.Assignment{
			ID:           assignmentID,
			DocumentID:   docID,
			DocumentKind: "incoming_letter",
			ExecutorID:   executorID,
			Content:      "Контент",
			Status:       "completed",
			Report:       "Отчет",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "completed", "Отчет")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, eventStore.created, 2)
		assert.ElementsMatch(t, []uuid.UUID{controllerID, executorID}, []uuid.UUID{
			eventStore.created[0].RecipientUserID,
			eventStore.created[1].RecipientUserID,
		})
		for _, event := range eventStore.created {
			assert.Equal(t, models.UserEventAssignmentCompleted, event.EventType)
			assert.Equal(t, "Поручение ожидает приемки", event.Title)
		}
	})

	t.Run("completed notifies assignee with assign access", func(t *testing.T) {
		svc, repo, userRepo, auth, _ := setupAssignmentService(t, "executor")
		executorID, _ := uuid.Parse(auth.GetCurrentUserID())
		svc.access.accessRepo = newKindActionAccessStore(map[string][]string{
			"incoming_letter": {"assign"},
		})
		eventStore := &fakeUserEventStore{}
		svc.events = NewUserEventService(eventStore, auth)
		userRepo.On("GetAll").Return([]models.User{
			{ID: executorID, IsActive: true},
		}, nil).Once()

		existing := &models.Assignment{
			ID:           assignmentID,
			DocumentID:   docID,
			DocumentKind: "incoming_letter",
			ExecutorID:   executorID,
			Content:      "Контент",
			Status:       "in_progress",
		}
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Update", assignmentID, executorID, "Контент",
			(*time.Time)(nil), "completed", "Отчет",
			mock.AnythingOfType("*time.Time"), []string(nil),
		).Return(&models.Assignment{
			ID:           assignmentID,
			DocumentID:   docID,
			DocumentKind: "incoming_letter",
			ExecutorID:   executorID,
			Content:      "Контент",
			Status:       "completed",
			Report:       "Отчет",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "completed", "Отчет")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, eventStore.created, 1)
		assert.Equal(t, executorID, eventStore.created[0].RecipientUserID)
		assert.Equal(t, models.UserEventAssignmentCompleted, eventStore.created[0].EventType)
	})

	t.Run("completed notifies coexecutor with assign access", func(t *testing.T) {
		svc, repo, userRepo, auth, _ := setupAssignmentService(t, "executor")
		executorID, _ := uuid.Parse(auth.GetCurrentUserID())
		coExecutorID := uuid.New()
		svc.access.accessRepo = newKindActionAccessStore(map[string][]string{
			"incoming_letter": {"assign"},
		})
		eventStore := &fakeUserEventStore{}
		svc.events = NewUserEventService(eventStore, auth)
		userRepo.On("GetAll").Return([]models.User{
			{ID: coExecutorID, IsActive: true},
			{ID: executorID, IsActive: true},
		}, nil).Once()

		existing := &models.Assignment{
			ID:            assignmentID,
			DocumentID:    docID,
			DocumentKind:  "incoming_letter",
			ExecutorID:    executorID,
			CoExecutorIDs: []string{coExecutorID.String()},
			Content:       "Контент",
			Status:        "in_progress",
		}
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Update", assignmentID, executorID, "Контент",
			(*time.Time)(nil), "completed", "Отчет",
			mock.AnythingOfType("*time.Time"), []string{coExecutorID.String()},
		).Return(&models.Assignment{
			ID:            assignmentID,
			DocumentID:    docID,
			DocumentKind:  "incoming_letter",
			ExecutorID:    executorID,
			CoExecutorIDs: []string{coExecutorID.String()},
			Content:       "Контент",
			Status:        "completed",
			Report:        "Отчет",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "completed", "Отчет")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, eventStore.created, 2)
		assert.ElementsMatch(t, []uuid.UUID{coExecutorID, executorID}, []uuid.UUID{
			eventStore.created[0].RecipientUserID,
			eventStore.created[1].RecipientUserID,
		})
	})

	t.Run("returned notifies executor", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "clerk")
		eventStore := &fakeUserEventStore{}
		svc.events = NewUserEventService(eventStore, auth)
		completedAt := time.Now()
		existing := &models.Assignment{
			ID:          assignmentID,
			DocumentID:  docID,
			ExecutorID:  execID,
			Content:     "Контент",
			Status:      "completed",
			Report:      "Отчет",
			CompletedAt: &completedAt,
		}
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Update", assignmentID, execID, "Контент",
			(*time.Time)(nil), "returned", "Нужно доработать",
			(*time.Time)(nil), []string(nil),
		).Return(&models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "returned",
			Report:     "Нужно доработать",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "returned", "Нужно доработать")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, eventStore.created, 1)
		assert.Equal(t, execID, eventStore.created[0].RecipientUserID)
		assert.Equal(t, models.UserEventAssignmentReturned, eventStore.created[0].EventType)
	})
}

// ---------- TestAssignmentService_GetByID ----------

func TestAssignmentService_GetByID(t *testing.T) {
	// Получение полной информации о поручении по его ID
	assignmentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		expected := &models.Assignment{
			ID:      assignmentID,
			Content: "Контент",
			Status:  "new",
		}
		repo.On("GetByID", assignmentID).Return(expected, nil).Once()

		result, err := svc.GetByID(assignmentID.String())
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, assignmentID.String(), result.ID)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc, _ := setupAssignmentServiceNotAuth(t)

		result, err := svc.GetByID(assignmentID.String())
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _, _, _, _ := setupAssignmentService(t, "executor")

		result, err := svc.GetByID("not-a-uuid")
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID поручения")
		assert.Nil(t, result)
	})

	t.Run("admin forbidden", func(t *testing.T) {
		svc, _, _, _, _ := setupAssignmentService(t, "admin")
		result, err := svc.GetByID(assignmentID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		repo.On("GetByID", assignmentID).Return(nil, nil).Once()

		result, err := svc.GetByID(assignmentID.String())
		require.Error(t, err)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", appErr.Kind)
		assert.Equal(t, 404, appErr.Code)
		assert.Nil(t, result)
	})
}

// ---------- TestAssignmentService_GetList ----------

func TestAssignmentService_GetList(t *testing.T) {
	// Получение списка поручений с фильтрацией (для дашборда или списков)
	t.Run("success", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "executor")

		filter := models.AssignmentFilter{Page: 1, PageSize: 20}
		filter.ExecutorID = auth.GetCurrentUserID()
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		}
		repo.On("GetList", filter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, 1, result.TotalCount)
	})

	t.Run("default pagination", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "executor")

		// Filter with 0 page/pagesize — should default to 1/20
		filter := models.AssignmentFilter{Page: 0, PageSize: 0}
		expectedFilter := models.AssignmentFilter{Page: 1, PageSize: 20, ExecutorID: auth.GetCurrentUserID()}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{},
			TotalCount: 0,
			Page:       1,
			PageSize:   20,
		}
		repo.On("GetList", expectedFilter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc, _ := setupAssignmentServiceNotAuth(t)

		result, err := svc.GetList(models.AssignmentFilter{})
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})

	t.Run("admin without document rights is forbidden", func(t *testing.T) {
		svc, _, _, _, _ := setupAssignmentService(t, "admin")
		result, err := svc.GetList(models.AssignmentFilter{})
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("document assignments allowed for clerk", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		docID := uuid.New()
		filter := models.AssignmentFilter{DocumentID: docID.String(), Page: 1, PageSize: 100, ShowFinished: true}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), DocumentID: docID, Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   100,
		}
		repo.On("GetList", filter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
	})

	t.Run("document assignments scoped for executor without assign right", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "executor")
		docID := uuid.New()
		filter := models.AssignmentFilter{
			DocumentID:   docID.String(),
			Page:         1,
			PageSize:     100,
			ShowFinished: true,
		}
		expectedFilter := models.AssignmentFilter{
			DocumentID:         docID.String(),
			Page:               1,
			PageSize:           100,
			ShowFinished:       true,
			ExecutorID:         auth.GetCurrentUserID(),
			AccessibleByUserID: auth.GetCurrentUserID(),
		}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), DocumentID: docID, ExecutorID: uuid.MustParse(auth.GetCurrentUserID()), Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   100,
		}
		repo.On("GetList", expectedFilter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
	})

	t.Run("partial assignment rights are scoped by document kind and own assignments", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "executor")
		svc.access = NewDocumentAccessService(
			auth,
			nil,
			repo,
			nil,
			newKindActionAccessStore(map[string][]string{
				string(models.DocumentKindIncomingLetter): {"assign"},
			}),
			nil,
			nil,
			nil,
		)

		filter := models.AssignmentFilter{Page: 1, PageSize: 20}
		expectedFilter := models.AssignmentFilter{
			Page:                 1,
			PageSize:             20,
			AllowedDocumentKinds: []string{string(models.DocumentKindIncomingLetter)},
			AccessibleByUserID:   auth.GetCurrentUserID(),
		}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		}
		repo.On("GetList", expectedFilter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
	})
}

// ---------- TestAssignmentService_Delete ----------

func TestAssignmentService_Delete(t *testing.T) {
	// Удаление поручения из системы
	assignmentID := uuid.New()
	execID := uuid.New()
	docID := uuid.New()

	existing := &models.Assignment{
		ID:         assignmentID,
		DocumentID: docID,
		ExecutorID: execID,
		Content:    "Контент",
		Status:     "new",
	}

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Delete", assignmentID).Return(nil).Once()

		err := svc.Delete(assignmentID.String())
		require.NoError(t, err)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "executor")
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()

		err := svc.Delete(assignmentID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("finished not admin", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		finished := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "finished",
		}

		repo.On("GetByID", assignmentID).Return(finished, nil).Once()

		err := svc.Delete(assignmentID.String())
		require.Error(t, err)
		requireAppError(t, err, "CONFLICT", 409, "завершённое")
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		repo.On("GetByID", assignmentID).Return(nil, nil).Once()

		err := svc.Delete(assignmentID.String())
		require.Error(t, err)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", appErr.Kind)
		assert.Equal(t, 404, appErr.Code)
	})
}

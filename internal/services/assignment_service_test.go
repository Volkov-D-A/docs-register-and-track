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
	*AssignmentService, *mocks.AssignmentStore, *mocks.UserStore, *AuthService,
) {
	t.Helper()
	assignmentRepo := mocks.NewAssignmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_user",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{role},
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth)

	svc := NewAssignmentService(assignmentRepo, userRepo, auth, journalSvc)
	return svc, assignmentRepo, userRepo, auth
}

func setupAssignmentServiceNotAuth(t *testing.T) (*AssignmentService, *mocks.AssignmentStore) {
	t.Helper()
	assignmentRepo := mocks.NewAssignmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	journalRepo := mocks.NewJournalStore(t)
	journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
	journalSvc := NewJournalService(journalRepo, auth)

	svc := NewAssignmentService(assignmentRepo, userRepo, auth, journalSvc)
	return svc, assignmentRepo
}

// ---------- TestAssignmentService_Create ----------

func TestAssignmentService_Create(t *testing.T) {
	// Создание нового поручения по документу
	docID := uuid.New()
	execID := uuid.New()

	t.Run("success with deadline", func(t *testing.T) {
		svc, repo, _, _ := setupAssignmentService(t, "clerk")

		expected := &models.Assignment{
			ID:         uuid.New(),
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Выполнить",
			Status:     "new",
		}
		repo.On("Create", docID, "incoming", execID, "Выполнить",
			mock.AnythingOfType("*time.Time"), []string(nil),
		).Return(expected, nil).Once()

		result, err := svc.Create(docID.String(), "incoming", execID.String(), "Выполнить", "2025-12-31", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expected.ID.String(), result.ID)
	})

	t.Run("success without deadline", func(t *testing.T) {
		svc, repo, _, _ := setupAssignmentService(t, "clerk")

		expected := &models.Assignment{
			ID:         uuid.New(),
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Выполнить",
			Status:     "new",
		}
		repo.On("Create", docID, "incoming", execID, "Выполнить",
			(*time.Time)(nil), []string(nil),
		).Return(expected, nil).Once()

		result, err := svc.Create(docID.String(), "incoming", execID.String(), "Выполнить", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc, _ := setupAssignmentServiceNotAuth(t)

		result, err := svc.Create(docID.String(), "incoming", execID.String(), "Выполнить", "", nil)
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})

	t.Run("invalid document ID", func(t *testing.T) {
		svc, _, _, _ := setupAssignmentService(t, "clerk")

		result, err := svc.Create("not-a-uuid", "incoming", execID.String(), "Выполнить", "", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document ID")
		assert.Nil(t, result)
	})

	t.Run("invalid executor ID", func(t *testing.T) {
		svc, _, _, _ := setupAssignmentService(t, "clerk")

		result, err := svc.Create(docID.String(), "incoming", "not-a-uuid", "Выполнить", "", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid executor ID")
		assert.Nil(t, result)
	})
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
		svc, repo, _, _ := setupAssignmentService(t, "clerk")

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
		svc, repo, _, _ := setupAssignmentService(t, "clerk")

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
		svc, _, _, _ := setupAssignmentService(t, "executor")

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(nil, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не найдено")
		assert.Nil(t, result)
	})

	t.Run("finished not admin", func(t *testing.T) {
		svc, repo, _, _ := setupAssignmentService(t, "clerk")
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
		assert.Contains(t, err.Error(), "завершённое")
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
		_, repo, userRepo, _ := setupAssignmentService(t, "executor")
		// Override auth currentUser ID to match executor
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		executorUser := &models.User{
			ID:           execID,
			Login:        "exec",
			PasswordHash: hash,
			IsActive:     true,
			Roles:        []string{"executor"},
		}
		authSvc := NewAuthService(nil, userRepo)
		userRepo.On("GetByLogin", "exec").Return(executorUser, nil).Once()
		authSvc.Login("exec", password)
		userRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()
		journalRepo := mocks.NewJournalStore(t)
		journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
		journalSvc := NewJournalService(journalRepo, authSvc)

		svc2 := NewAssignmentService(repo, userRepo, authSvc, journalSvc)

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
		_, repo, userRepo, _ := setupAssignmentService(t, "executor")
		password := "Passw0rd!"
		hash, _ := security.HashPassword(password)
		executorUser := &models.User{
			ID:           execID,
			Login:        "exec2",
			PasswordHash: hash,
			IsActive:     true,
			Roles:        []string{"executor"},
		}
		authSvc := NewAuthService(nil, userRepo)
		userRepo.On("GetByLogin", "exec2").Return(executorUser, nil).Once()
		authSvc.Login("exec2", password)
		userRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()
		journalRepo := mocks.NewJournalStore(t)
		journalRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.Nil, nil).Maybe()
		journalSvc := NewJournalService(journalRepo, authSvc)

		svc2 := NewAssignmentService(repo, userRepo, authSvc, journalSvc)

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
		svc, repo, _, _ := setupAssignmentService(t, "clerk")
		completedAssignment := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "completed",
		}

		repo.On("GetByID", assignmentID).Return(completedAssignment, nil).Once()
		repo.On("Update", assignmentID, execID, "Контент",
			(*time.Time)(nil), "finished", "",
			(*time.Time)(nil), []string(nil),
		).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "finished",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "finished", result.Status)
	})

	t.Run("admin forbidden", func(t *testing.T) {
		svc, repo, _, _ := setupAssignmentService(t, "admin")
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "недостаточно прав")
		assert.Nil(t, result)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, repo, _, _ := setupAssignmentService(t, "executor")
		// Executor with different ID than assignment executor — can't set "finished"
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// ---------- TestAssignmentService_GetByID ----------

func TestAssignmentService_GetByID(t *testing.T) {
	// Получение полной информации о поручении по его ID
	assignmentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _ := setupAssignmentService(t, "clerk")

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
		svc, _, _, _ := setupAssignmentService(t, "executor")

		result, err := svc.GetByID("not-a-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ID")
		assert.Nil(t, result)
	})

	t.Run("admin forbidden", func(t *testing.T) {
		svc, _, _, _ := setupAssignmentService(t, "admin")
		result, err := svc.GetByID(assignmentID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})
}

// ---------- TestAssignmentService_GetList ----------

func TestAssignmentService_GetList(t *testing.T) {
	// Получение списка поручений с фильтрацией (для дашборда или списков)
	t.Run("success", func(t *testing.T) {
		svc, repo, _, auth := setupAssignmentService(t, "executor")

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
		svc, repo, _, auth := setupAssignmentService(t, "executor")

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

	t.Run("admin forbidden", func(t *testing.T) {
		svc, _, _, _ := setupAssignmentService(t, "admin")
		result, err := svc.GetList(models.AssignmentFilter{})
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
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
		svc, repo, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("Delete", assignmentID).Return(nil).Once()

		err := svc.Delete(assignmentID.String())
		require.NoError(t, err)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _, _, _ := setupAssignmentService(t, "executor")

		err := svc.Delete(assignmentID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("finished not admin", func(t *testing.T) {
		svc, repo, _, _ := setupAssignmentService(t, "clerk")
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
		assert.Contains(t, err.Error(), "завершённое")
	})
}

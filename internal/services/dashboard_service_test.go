package services

import (
	"context"
	"testing"
	"time"

	"docflow/internal/mocks"
	"docflow/internal/models"
	"docflow/internal/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockStorageInfo — мок для StorageInfoProvider.
type mockStorageInfo struct {
	objectCount int
	totalSize   string
	err         error
}

func (m *mockStorageInfo) GetStorageInfo(ctx context.Context) (int, string, error) {
	return m.objectCount, m.totalSize, m.err
}

func TestDashboardService_GetStats(t *testing.T) {
	// Получение сводной статистики для рабочего стола (индивидуально для каждой роли: executor, admin, clerk)
	mockRepo := mocks.NewDashboardStore(t)
	authRepo := mocks.NewUserStore(t)
	authService := NewAuthService(nil, authRepo)
	mockStorage := &mockStorageInfo{objectCount: 42, totalSize: "128.5 MB"}
	dashboardService := NewDashboardService(mockRepo, authService, mockStorage)

	login := "testuser"
	password := "CorrectPassw0rd!"
	hash, _ := security.HashPassword(password)

	executorUser := &models.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{"executor"},
	}

	adminUser := &models.User{
		ID:           uuid.New(),
		Login:        "adminuser",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{"admin"},
	}

	clerkUser := &models.User{
		ID:           uuid.New(),
		Login:        "clerkuser",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{"clerk"},
	}

	t.Run("executor stats", func(t *testing.T) {
		authRepo.On("GetByLogin", login).Return(executorUser, nil).Once()
		authService.Login(login, password)

		mockRepo.On("GetExecutorStatusCounts", executorUser.ID).Return(5, 2, nil).Once()
		mockRepo.On("GetExecutorOverdueCount", executorUser.ID).Return(1, nil).Once()
		mockRepo.On("GetExecutorFinishedCounts", executorUser.ID).Return(10, 3, nil).Once()

		var assignments []models.Assignment
		uidPtr := &executorUser.ID
		mockRepo.On("GetExpiringAssignments", uidPtr, 3).Return(assignments, nil).Once()

		stats, err := dashboardService.GetStats("", "", "")
		require.NoError(t, err)
		assert.Equal(t, "executor", stats.Role)
		assert.Equal(t, 5, stats.MyAssignmentsNew)
		assert.Equal(t, 10, stats.MyAssignmentsFinished)

		authService.Logout()
	})

	t.Run("admin stats", func(t *testing.T) {
		authRepo.On("GetByLogin", "adminuser").Return(adminUser, nil).Once()
		authService.Login("adminuser", password)

		mockRepo.On("GetAdminUserCount").Return(50, nil).Once()
		mockRepo.On("GetAdminDocCounts").Return(100, 200, nil).Once()
		mockRepo.On("GetDBSize").Return("10MB").Once()

		stats, err := dashboardService.GetStats("", "", "")
		require.NoError(t, err)
		assert.Equal(t, "admin", stats.Role)
		assert.Equal(t, 50, stats.UserCount)
		assert.Equal(t, 300, stats.TotalDocuments)
		assert.Equal(t, "10MB", stats.DBSize)
		assert.Equal(t, 42, stats.StorageObjects)
		assert.Equal(t, "128.5 MB", stats.StorageSize)

		authService.Logout()
	})

	t.Run("clerk stats", func(t *testing.T) {
		authRepo.On("GetByLogin", "clerkuser").Return(clerkUser, nil).Once()
		authService.Login("clerkuser", password)

		// Тестирование с конкретными датами
		startDateStr := "2024-01-01"
		endDateStr := "2024-01-31"

		start, _ := time.Parse("2006-01-02", startDateStr)
		endParsed, _ := time.Parse("2006-01-02", endDateStr)
		end := endParsed.Add(24*time.Hour - time.Nanosecond)

		mockRepo.On("GetDocCountsByPeriod", start, end).Return(50, 40, nil).Once()
		mockRepo.On("GetOverdueCountByPeriod", start, end).Return(5, nil).Once()
		mockRepo.On("GetFinishedCountsByPeriod", start, end).Return(80, 10, nil).Once()

		var assignments []models.Assignment
		var uidPtr *uuid.UUID = nil
		mockRepo.On("GetExpiringAssignments", uidPtr, 7).Return(assignments, nil).Once()

		stats, err := dashboardService.GetStats("", startDateStr, endDateStr)
		require.NoError(t, err)
		assert.Equal(t, "clerk", stats.Role)
		assert.Equal(t, 50, stats.IncomingCount)
		assert.Equal(t, 40, stats.OutgoingCount)
		assert.Equal(t, 5, stats.AllAssignmentsOverdue)

		authService.Logout()
	})

	t.Run("not authenticated", func(t *testing.T) {
		stats, err := dashboardService.GetStats("", "", "")
		require.ErrorIs(t, err, ErrNotAuthenticated)
		require.Nil(t, stats)
	})

	t.Run("clerk invalid start date", func(t *testing.T) {
		authRepo.On("GetByLogin", "clerkuser").Return(clerkUser, nil).Once()
		authService.Login("clerkuser", password)

		stats, err := dashboardService.GetStats("", "invalid-date", "2024-01-31")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid start date")
		require.Nil(t, stats)

		authService.Logout()
	})

	t.Run("clerk invalid end date", func(t *testing.T) {
		authRepo.On("GetByLogin", "clerkuser").Return(clerkUser, nil).Once()
		authService.Login("clerkuser", password)

		stats, err := dashboardService.GetStats("", "2024-01-01", "invalid-date")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid end date")
		require.Nil(t, stats)

		authService.Logout()
	})

	t.Run("executor specific role fallback and error", func(t *testing.T) {
		authRepo.On("GetByLogin", login).Return(executorUser, nil).Once()
		authService.Login(login, password)

		// Ask for admin role, but user doesn't have it, so it falls back to executor
		mockRepo.On("GetExecutorStatusCounts", executorUser.ID).Return(0, 0, assert.AnError).Once()

		stats, err := dashboardService.GetStats("admin", "", "")
		require.ErrorIs(t, err, assert.AnError)
		require.Nil(t, stats)

		authService.Logout()
	})

	t.Run("admin missing user count error", func(t *testing.T) {
		authRepo.On("GetByLogin", "adminuser").Return(adminUser, nil).Once()
		authService.Login("adminuser", password)

		mockRepo.On("GetAdminUserCount").Return(0, assert.AnError).Once()

		stats, err := dashboardService.GetStats("", "", "")
		require.ErrorIs(t, err, assert.AnError)
		require.Nil(t, stats)

		authService.Logout()
	})

	t.Run("clerk GetDocCountsByPeriod error", func(t *testing.T) {
		authRepo.On("GetByLogin", "clerkuser").Return(clerkUser, nil).Once()
		authService.Login("clerkuser", password)

		// Test empty dates (defaulting to current month)
		mockRepo.On("GetDocCountsByPeriod", time.Time{}, time.Time{}).Return(0, 0, assert.AnError).Maybe()
		// Depending on time.Now(), we just catch the first call pattern
		mockRepo.Calls = nil
		mockRepo.ExpectedCalls = nil

		// Catch any GetDocCountsByPeriod call and return error
		mockRepo.On("GetDocCountsByPeriod", mock.Anything, mock.Anything).Return(0, 0, assert.AnError).Once()

		stats, err := dashboardService.GetStats("", "", "")
		require.ErrorIs(t, err, assert.AnError)
		require.Nil(t, stats)

		authService.Logout()
	})
}

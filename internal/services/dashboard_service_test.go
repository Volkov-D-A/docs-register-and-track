package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"

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

func contains(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
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
	makeDashboardService := func(accessRoles ...string) *DashboardService {
		if contains(accessRoles, "clerk") {
			accessRoles = append(accessRoles,
				models.SystemPermissionStatsIncoming,
				models.SystemPermissionStatsOutgoing,
				models.SystemPermissionStatsAssignments,
			)
		}
		if contains(accessRoles, "executor") {
			accessRoles = append(accessRoles, models.SystemPermissionStatsAssignments)
		}
		if contains(accessRoles, "admin") {
			accessRoles = append(accessRoles, models.SystemPermissionStatsSystem)
		}
		accessStore := newRoleMappedDocumentAccessStore(accessRoles...)
		authService.SetAccessStore(accessStore)
		accessService := NewDocumentAccessService(authService, nil, nil, nil, accessStore, nil, nil, nil)
		return NewDashboardService(mockRepo, authService, mockStorage, accessService)
	}

	login := "testuser"
	password := "CorrectPassw0rd!"
	hash, _ := security.HashPassword(password)

	executorUser := &models.User{
		ID:                    uuid.New(),
		Login:                 login,
		PasswordHash:          hash,
		IsDocumentParticipant: true,
		IsActive:              true,
	}

	adminUser := &models.User{
		ID:                    uuid.New(),
		Login:                 "adminuser",
		PasswordHash:          hash,
		IsDocumentParticipant: false,
		IsActive:              true,
		SystemPermissions:     []string{models.SystemPermissionAdmin},
	}

		clerkUser := &models.User{
			ID:                    uuid.New(),
			Login:                 "clerkuser",
			PasswordHash:          hash,
			IsDocumentParticipant: false,
			IsActive:              true,
		}

	mixedUser := &models.User{
		ID:                    uuid.New(),
		Login:                 "mixeduser",
		PasswordHash:          hash,
		IsDocumentParticipant: true,
		IsActive:              true,
	}

	t.Run("executor stats", func(t *testing.T) {
		dashboardService := makeDashboardService("executor")
		authRepo.On("GetByLogin", login).Return(executorUser, nil).Once()
		authService.Login(login, password)
		authRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()

		mockRepo.On("GetExecutorStatusCounts", executorUser.ID).Return(5, 2, nil).Once()
		mockRepo.On("GetExecutorOverdueCount", executorUser.ID).Return(1, nil).Once()
		mockRepo.On("GetExecutorFinishedCounts", executorUser.ID).Return(10, 3, nil).Once()

		var assignments []models.Assignment
		uidPtr := &executorUser.ID
		mockRepo.On("GetExpiringAssignments", uidPtr, 3).Return(assignments, nil).Once()

			stats, err := dashboardService.GetStats("", "", "")
			require.NoError(t, err)
			assert.Equal(t, 5, stats.MyAssignmentsNew)
			assert.Equal(t, 10, stats.MyAssignmentsFinished)

		authService.Logout()
	})

	t.Run("admin stats", func(t *testing.T) {
		dashboardService := makeDashboardService("admin")
		authRepo.On("GetByLogin", "adminuser").Return(adminUser, nil).Once()
		authService.Login("adminuser", password)
		authRepo.On("GetByID", adminUser.ID).Return(adminUser, nil).Maybe()

		mockRepo.On("GetAdminUserCount").Return(50, nil).Once()
		mockRepo.On("GetAdminDocCounts").Return(100, 200, nil).Once()
		mockRepo.On("GetDBSize").Return("10MB").Once()

			stats, err := dashboardService.GetStats("", "", "")
			require.NoError(t, err)
			assert.Equal(t, 50, stats.UserCount)
			assert.Equal(t, 300, stats.TotalDocuments)
		assert.Equal(t, "10MB", stats.DBSize)
		assert.Equal(t, 42, stats.StorageObjects)
		assert.Equal(t, "128.5 MB", stats.StorageSize)

		authService.Logout()
	})

	t.Run("clerk stats", func(t *testing.T) {
		dashboardService := makeDashboardService("clerk")
		authRepo.On("GetByLogin", "clerkuser").Return(clerkUser, nil).Once()
		authService.Login("clerkuser", password)
		authRepo.On("GetByID", clerkUser.ID).Return(clerkUser, nil).Maybe()

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
			assert.Equal(t, 50, stats.IncomingCount)
			assert.Equal(t, 40, stats.OutgoingCount)
		assert.Equal(t, 5, stats.AllAssignmentsOverdue)

		authService.Logout()
	})

	t.Run("not authenticated", func(t *testing.T) {
		dashboardService := makeDashboardService()
		stats, err := dashboardService.GetStats("", "", "")
		require.ErrorIs(t, err, ErrNotAuthenticated)
		require.Nil(t, stats)
	})

	t.Run("clerk invalid start date", func(t *testing.T) {
		dashboardService := makeDashboardService("clerk")
		authRepo.On("GetByLogin", "clerkuser").Return(clerkUser, nil).Once()
		authService.Login("clerkuser", password)
		authRepo.On("GetByID", clerkUser.ID).Return(clerkUser, nil).Maybe()

		stats, err := dashboardService.GetStats("", "invalid-date", "2024-01-31")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid start date")
		require.Nil(t, stats)

		authService.Logout()
	})

	t.Run("clerk invalid end date", func(t *testing.T) {
		dashboardService := makeDashboardService("clerk")
		authRepo.On("GetByLogin", "clerkuser").Return(clerkUser, nil).Once()
		authService.Login("clerkuser", password)
		authRepo.On("GetByID", clerkUser.ID).Return(clerkUser, nil).Maybe()

		stats, err := dashboardService.GetStats("", "2024-01-01", "invalid-date")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid end date")
		require.Nil(t, stats)

		authService.Logout()
	})

	t.Run("requested role is ignored in favor of computed profile", func(t *testing.T) {
		dashboardService := makeDashboardService("executor")
		authRepo.On("GetByLogin", login).Return(executorUser, nil).Once()
		authService.Login(login, password)
		authRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()

		var assignments []models.Assignment
		uidPtr := &executorUser.ID
		mockRepo.On("GetExecutorStatusCounts", executorUser.ID).Return(1, 1, nil).Once()
		mockRepo.On("GetExecutorOverdueCount", executorUser.ID).Return(0, nil).Once()
		mockRepo.On("GetExecutorFinishedCounts", executorUser.ID).Return(2, 0, nil).Once()
		mockRepo.On("GetExpiringAssignments", uidPtr, 3).Return(assignments, nil).Once()

			stats, err := dashboardService.GetStats("admin", "", "")
			require.NoError(t, err)
			require.NotNil(t, stats)

			authService.Logout()
	})

	t.Run("mixed stats", func(t *testing.T) {
		dashboardService := makeDashboardService("clerk", "executor")
		authRepo.On("GetByLogin", "mixeduser").Return(mixedUser, nil).Once()
		authService.Login("mixeduser", password)
		authRepo.On("GetByID", mixedUser.ID).Return(mixedUser, nil).Maybe()

		startDateStr := "2024-01-01"
		endDateStr := "2024-01-31"

		start, _ := time.Parse("2006-01-02", startDateStr)
		endParsed, _ := time.Parse("2006-01-02", endDateStr)
		end := endParsed.Add(24*time.Hour - time.Nanosecond)

			mockRepo.On("GetDocCountsByPeriod", start, end).Return(12, 8, nil).Once()
			mockRepo.On("GetOverdueCountByPeriod", start, end).Return(2, nil).Once()
			mockRepo.On("GetFinishedCountsByPeriod", start, end).Return(6, 1, nil).Once()
			var globalAssignments []models.Assignment
			mockRepo.On("GetExpiringAssignments", (*uuid.UUID)(nil), 7).Return(globalAssignments, nil).Once()
			mockRepo.On("GetExecutorStatusCounts", mixedUser.ID).Return(3, 4, nil).Once()
			mockRepo.On("GetExecutorOverdueCount", mixedUser.ID).Return(1, nil).Once()
			mockRepo.On("GetExecutorFinishedCounts", mixedUser.ID).Return(5, 1, nil).Once()
			var personalAssignments []models.Assignment
			mockRepo.On("GetExpiringAssignments", &mixedUser.ID, 3).Return(personalAssignments, nil).Once()

			stats, err := dashboardService.GetStats("", startDateStr, endDateStr)
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, 12, stats.IncomingCount)
			assert.Equal(t, 2, stats.AllAssignmentsOverdue)
			assert.Equal(t, 3, stats.MyAssignmentsNew)
			assert.Equal(t, 5, stats.MyAssignmentsFinished)

		authService.Logout()
	})

	t.Run("admin missing user count error", func(t *testing.T) {
		dashboardService := makeDashboardService("admin")
		authRepo.On("GetByLogin", "adminuser").Return(adminUser, nil).Once()
		authService.Login("adminuser", password)
		authRepo.On("GetByID", adminUser.ID).Return(adminUser, nil).Maybe()

		mockRepo.On("GetAdminUserCount").Return(0, assert.AnError).Once()

		stats, err := dashboardService.GetStats("", "", "")
		require.ErrorIs(t, err, assert.AnError)
		require.Nil(t, stats)

		authService.Logout()
	})

	t.Run("clerk GetDocCountsByPeriod error", func(t *testing.T) {
		dashboardService := makeDashboardService("clerk")
		authRepo.On("GetByLogin", "clerkuser").Return(clerkUser, nil).Once()
		authService.Login("clerkuser", password)
		authRepo.On("GetByID", clerkUser.ID).Return(clerkUser, nil).Maybe()

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

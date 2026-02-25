package services

import (
	"testing"
	"time"

	"docflow/internal/mocks"
	"docflow/internal/models"
	"docflow/internal/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardService_GetStats(t *testing.T) {
	mockRepo := mocks.NewDashboardStore(t)
	authRepo := mocks.NewUserStore(t)
	authService := NewAuthService(nil, authRepo)
	dashboardService := NewDashboardService(mockRepo, authService)

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
}

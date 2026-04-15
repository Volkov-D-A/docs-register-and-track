package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DashboardService предоставляет бизнес-логику для формирования данных дашборда.
type DashboardService struct {
	repo    DashboardStore
	auth    *AuthService
	storage StorageInfoProvider
}

// NewDashboardService создает новый экземпляр DashboardService.
func NewDashboardService(repo DashboardStore, auth *AuthService, storage StorageInfoProvider) *DashboardService {
	return &DashboardService{repo: repo, auth: auth, storage: storage}
}

func determineDashboardProfile(user *dto.User) string {
	if user == nil {
		return "executor"
	}

	hasRole := func(expected string) bool {
		for _, role := range user.Roles {
			if role == expected {
				return true
			}
		}
		return false
	}

	hasClerk := hasRole("clerk")
	hasExecutor := hasRole("executor")

	switch {
	case hasClerk && hasExecutor:
		return "mixed"
	case hasClerk:
		return "clerk"
	case hasExecutor:
		return "executor"
	case hasRole("admin"):
		return "admin"
	default:
		return "executor"
	}
}

// GetStats возвращает статистику для дашборда в зависимости от роли пользователя.
func (s *DashboardService) GetStats(requestedRole string, startDateStr, endDateStr string) (*dto.DashboardStats, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	_ = requestedRole

	role := determineDashboardProfile(user)

	// Инициализация пустым списком для избежания null в JSON
	stats := &models.DashboardStats{
		Role:                role,
		ExpiringAssignments: []models.Assignment{},
	}

	var result *models.DashboardStats
	switch role {
	case "admin":
		result, err = s.getAdminStats(stats)
	case "clerk":
		// Parse dates
		var startDate, endDate time.Time

		if startDateStr == "" || endDateStr == "" {
			// Default to current month if empty
			now := time.Now()
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			endDate = startDate.AddDate(0, 1, -1).Add(24*time.Hour - time.Nanosecond)
		} else {
			// Формат даты "2006-01-02"
			startDate, err = time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return nil, fmt.Errorf("invalid start date: %w", err)
			}
			// Для конечной даты устанавливаем 23:59:59, чтобы включить последний день
			endDateParsed, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return nil, fmt.Errorf("invalid end date: %w", err)
			}
			endDate = endDateParsed.Add(24*time.Hour - time.Nanosecond)
		}

		result, err = s.getClerkStats(stats, startDate, endDate)
	case "mixed":
		uid, _ := uuid.Parse(user.ID)
		result, err = s.getMixedStats(stats, uid, startDateStr, endDateStr)
	default:
		// Исполнитель (по умолчанию)
		uid, _ := uuid.Parse(user.ID)
		result, err = s.getExecutorStats(stats, uid)
	}

	if err != nil {
		return nil, err
	}

	return dto.MapDashboardStats(result), nil
}

func (s *DashboardService) getMixedStats(stats *models.DashboardStats, userID uuid.UUID, startDateStr, endDateStr string) (*models.DashboardStats, error) {
	var startDate, endDate time.Time
	var err error

	if startDateStr == "" || endDateStr == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, -1).Add(24*time.Hour - time.Nanosecond)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start date: %w", err)
		}
		endDateParsed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end date: %w", err)
		}
		endDate = endDateParsed.Add(24*time.Hour - time.Nanosecond)
	}

	stats, err = s.getClerkStats(stats, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return s.getExecutorStats(stats, userID)
}

func (s *DashboardService) getExecutorStats(stats *models.DashboardStats, userID uuid.UUID) (*models.DashboardStats, error) {
	// 1. Количество по статусам
	newCount, inProgressCount, err := s.repo.GetExecutorStatusCounts(userID)
	if err != nil {
		return nil, err
	}
	stats.MyAssignmentsNew = newCount
	stats.MyAssignmentsInProgress = inProgressCount

	// 2. Просроченные
	stats.MyAssignmentsOverdue, err = s.repo.GetExecutorOverdueCount(userID)
	if err != nil {
		return nil, err
	}

	// 3. Завершенные (всего) и завершенные с опозданием
	stats.MyAssignmentsFinished, stats.MyAssignmentsFinishedLate, err = s.repo.GetExecutorFinishedCounts(userID)
	if err != nil {
		return nil, err
	}

	// 4. Истекающие поручения (срок в течение 3 дней)
	assignments, err := s.repo.GetExpiringAssignments(&userID, 3)
	if err != nil {
		return nil, err
	}
	if assignments != nil {
		stats.ExpiringAssignments = assignments
	}

	return stats, nil
}

func (s *DashboardService) getClerkStats(stats *models.DashboardStats, startDate, endDate time.Time) (*models.DashboardStats, error) {
	// 1. Количество документов за период
	incoming, outgoing, err := s.repo.GetDocCountsByPeriod(startDate, endDate)
	if err != nil {
		return nil, err
	}
	stats.IncomingCount = incoming
	stats.OutgoingCount = outgoing

	// 2. Просроченные поручения за период
	stats.AllAssignmentsOverdue, err = s.repo.GetOverdueCountByPeriod(startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 3. Завершенные за период и завершенные с опозданием
	stats.AllAssignmentsFinished, stats.AllAssignmentsFinishedLate, err = s.repo.GetFinishedCountsByPeriod(startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 4. Истекающие поручения (глобально) — интервал 7 дней для делопроизводителей
	assignments, err := s.repo.GetExpiringAssignments(nil, 7)
	if err != nil {
		return nil, err
	}
	if assignments != nil {
		stats.ExpiringAssignments = assignments
	}

	return stats, nil
}

func (s *DashboardService) getAdminStats(stats *models.DashboardStats) (*models.DashboardStats, error) {
	// 1. Количество пользователей
	userCount, err := s.repo.GetAdminUserCount()
	if err != nil {
		return nil, err
	}
	stats.UserCount = userCount

	// 2. Всего документов
	inc, out, err := s.repo.GetAdminDocCounts()
	if err != nil {
		return nil, err
	}
	stats.TotalDocuments = inc + out

	// 3. Размер БД (PostgreSQL)
	stats.DBSize = s.repo.GetDBSize()

	// 4. Информация о файловом хранилище MinIO
	if s.storage != nil {
		objCount, totalSize, err := s.storage.GetStorageInfo(context.Background())
		if err != nil {
			slog.Warn("failed to get MinIO storage info", "error", err)
			stats.StorageSize = "N/A"
		} else {
			stats.StorageObjects = objCount
			stats.StorageSize = totalSize
		}
	} else {
		stats.StorageSize = "N/A"
	}

	return stats, nil
}

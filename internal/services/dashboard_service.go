package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/dto"
	"docflow/internal/models"
)

type DashboardService struct {
	repo DashboardStore
	auth *AuthService
}

func NewDashboardService(repo DashboardStore, auth *AuthService) *DashboardService {
	return &DashboardService{repo: repo, auth: auth}
}

func (s *DashboardService) GetStats(requestedRole string, startDateStr, endDateStr string) (*dto.DashboardStats, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	// Определение эффективной роли
	role := "executor"

	// Если запрошена конкретная роль, проверяем наличие у пользователя
	if requestedRole != "" {
		if s.auth.HasRole(requestedRole) {
			role = requestedRole
		} else {
			// Фолбэк на иерархию ролей по умолчанию
			if s.auth.HasRole("admin") {
				role = "admin"
			} else if s.auth.HasRole("clerk") {
				role = "clerk"
			}
		}
	} else {
		// Иерархия ролей по умолчанию, если роль не запрошена
		if s.auth.HasRole("admin") {
			role = "admin"
		} else if s.auth.HasRole("clerk") {
			role = "clerk"
		}
	}

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

	return stats, nil
}

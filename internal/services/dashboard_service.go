package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/database"
	"docflow/internal/dto"
	"docflow/internal/models"
)

type DashboardService struct {
	db   *database.DB
	auth *AuthService
}

func NewDashboardService(db *database.DB, auth *AuthService) *DashboardService {
	return &DashboardService{db: db, auth: auth}
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
	err := s.db.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'new'),
			COUNT(*) FILTER (WHERE status = 'in_progress')
		FROM assignments a
		WHERE executor_id = $1
		OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $1)
	`, userID).Scan(&stats.MyAssignmentsNew, &stats.MyAssignmentsInProgress)
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	// 2. Просроченные (status in ('new', 'in_progress') AND deadline < NOW())
	// ИЛИ status = 'completed' AND completed_at::date > deadline
	err = s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM assignments a
		WHERE (executor_id = $1 OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $1))
		  AND (
		      (status IN ('new', 'in_progress') AND deadline < CURRENT_DATE)
		      OR
		      (status = 'completed' AND completed_at::date > deadline)
		  )
	`, userID).Scan(&stats.MyAssignmentsOverdue)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue count: %w", err)
	}

	// 2.1 Завершенные (всего) и завершенные с опозданием
	err = s.db.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'finished'),
			COUNT(*) FILTER (WHERE status = 'finished' AND completed_at::date > deadline)
		FROM assignments a
		WHERE (executor_id = $1 OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $1))
	`, userID).Scan(&stats.MyAssignmentsFinished, &stats.MyAssignmentsFinishedLate)
	if err != nil {
		return nil, fmt.Errorf("failed to get finished counts: %w", err)
	}

	// 3. Истекающие поручения (срок в течение 3 дней)
	// Только активные поручения
	rows, err := s.db.Query(`
		SELECT 
			a.id, a.content, a.deadline, a.status,
			a.document_id, a.document_type,
			u.full_name as executor_name,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number
		FROM assignments a
		LEFT JOIN users u ON a.executor_id = u.id
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
		WHERE (a.executor_id = $1 OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $1))
		  AND a.status IN ('new', 'in_progress')
		  AND a.deadline BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '3 days')
		ORDER BY a.deadline ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring assignments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Assignment
		var docNumber sql.NullString
		var deadline sql.NullTime
		var executorName sql.NullString

		if err := rows.Scan(&a.ID, &a.Content, &deadline, &a.Status, &a.DocumentID, &a.DocumentType, &executorName, &docNumber); err != nil {
			return nil, err
		}
		if deadline.Valid {
			a.Deadline = &deadline.Time
		}
		if docNumber.Valid {
			a.DocumentNumber = docNumber.String
		}
		if executorName.Valid {
			a.ExecutorName = executorName.String
		}

		stats.ExpiringAssignments = append(stats.ExpiringAssignments, a)
	}

	return stats, nil
}

func (s *DashboardService) getClerkStats(stats *models.DashboardStats, startDate, endDate time.Time) (*models.DashboardStats, error) {
	// 1. Количество документов за период
	err := s.db.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM incoming_documents WHERE created_at BETWEEN $1 AND $2),
			(SELECT COUNT(*) FROM outgoing_documents WHERE created_at BETWEEN $1 AND $2)
	`, startDate, endDate).Scan(&stats.IncomingCount, &stats.OutgoingCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get doc counts: %w", err)
	}

	// 2. Просроченные поручения
	// Поручения со сроком в указанном периоде, которые просрочены
	err = s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM assignments 
		WHERE deadline BETWEEN $1 AND $2 
		  AND (
		      (status IN ('new', 'in_progress') AND deadline < CURRENT_DATE)
		      OR 
		      (status = 'completed' AND completed_at::date > deadline)
		  )
	`, startDate, endDate).Scan(&stats.AllAssignmentsOverdue)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue count: %w", err)
	}

	// 3. Завершенные за период и завершенные с опозданием
	// Фолбэк на updated_at, если completed_at == NULL (для старых данных)
	err = s.db.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'finished'),
			COUNT(*) FILTER (WHERE status = 'finished' AND COALESCE(completed_at, updated_at)::date > deadline)
		FROM assignments
		WHERE status = 'finished' AND COALESCE(completed_at, updated_at) BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&stats.AllAssignmentsFinished, &stats.AllAssignmentsFinishedLate)
	if err != nil {
		return nil, fmt.Errorf("failed to get all finished counts: %w", err)
	}

	// 4. Истекающие поручения (глобально) — интервал 7 дней для делопроизводителей
	// Список истекающих — всегда от текущей даты, не зависит от выбранного периода
	rows, err := s.db.Query(`
		SELECT 
			a.id, a.content, a.deadline, a.status,
			a.document_id, a.document_type,
			u.full_name as executor_name,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number
		FROM assignments a
		LEFT JOIN users u ON a.executor_id = u.id
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
		WHERE a.status IN ('new', 'in_progress')
		  AND a.deadline BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '7 days')
		ORDER BY a.deadline ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring assignments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Assignment
		var docNumber sql.NullString
		var deadline sql.NullTime
		var executorName sql.NullString

		if err := rows.Scan(&a.ID, &a.Content, &deadline, &a.Status, &a.DocumentID, &a.DocumentType, &executorName, &docNumber); err != nil {
			return nil, err
		}
		if deadline.Valid {
			a.Deadline = &deadline.Time
		}
		if docNumber.Valid {
			a.DocumentNumber = docNumber.String
		}
		if executorName.Valid {
			a.ExecutorName = executorName.String
		}

		stats.ExpiringAssignments = append(stats.ExpiringAssignments, a)
	}

	return stats, nil
}

func (s *DashboardService) getAdminStats(stats *models.DashboardStats) (*models.DashboardStats, error) {
	// 1. Количество пользователей
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.UserCount)
	if err != nil {
		return nil, err
	}

	// 2. Всего документов
	var inc, out int
	err = s.db.QueryRow("SELECT (SELECT COUNT(*) FROM incoming_documents), (SELECT COUNT(*) FROM outgoing_documents)").Scan(&inc, &out)
	if err != nil {
		return nil, err
	}
	stats.TotalDocuments = inc + out

	// 3. Размер БД (PostgreSQL)
	err = s.db.QueryRow("SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&stats.DBSize)
	if err != nil {
		stats.DBSize = "N/A"
	}

	return stats, nil
}

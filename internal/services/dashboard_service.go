package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/database"
	"docflow/internal/models"
)

type DashboardService struct {
	ctx  context.Context
	db   *database.DB
	auth *AuthService
}

func NewDashboardService(db *database.DB, auth *AuthService) *DashboardService {
	return &DashboardService{db: db, auth: auth}
}

func (s *DashboardService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *DashboardService) GetStats(requestedRole string, startDateStr, endDateStr string) (*models.DashboardStats, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	// Determine effective role
	role := "executor"

	// If a specific role is requested, verify user has it
	if requestedRole != "" {
		if s.auth.HasRole(requestedRole) {
			role = requestedRole
		} else {
			// Fallback or error? Let's fallback to default hierarchy for now to be safe
			if s.auth.HasRole("admin") {
				role = "admin"
			} else if s.auth.HasRole("clerk") {
				role = "clerk"
			}
		}
	} else {
		// Default hierarchy if no role requested
		if s.auth.HasRole("admin") {
			role = "admin"
		} else if s.auth.HasRole("clerk") {
			role = "clerk"
		}
	}

	// Initialize with empty slice to avoid null in JSON
	stats := &models.DashboardStats{
		Role:                role,
		ExpiringAssignments: []models.Assignment{},
	}

	switch role {
	case "admin":
		return s.getAdminStats(stats)
	case "clerk":

		// Parse dates
		var startDate, endDate time.Time
		var err error

		if startDateStr == "" || endDateStr == "" {
			// Default to current month if empty
			now := time.Now()
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			endDate = startDate.AddDate(0, 1, -1).Add(24*time.Hour - time.Nanosecond)
		} else {
			// Assume format "2006-01-02"
			startDate, err = time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return nil, fmt.Errorf("invalid start date: %w", err)
			}
			// For end date, we want the end of that day, so parse and add almost 24h or just compare date part in SQL
			// Let's parse as provided and assume it's 00:00:00, so we might want to set it to 23:59:59 if it's inclusive
			// Or strictly if the frontend sends "2023-01-01" to "2023-01-31", we want up to 2023-01-31 23:59:59.
			endDateParsed, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return nil, fmt.Errorf("invalid end date: %w", err)
			}
			endDate = endDateParsed.Add(24*time.Hour - time.Nanosecond)
		}

		return s.getClerkStats(stats, startDate, endDate)
	default:
		// Executor (default)
		return s.getExecutorStats(stats, user.ID)
	}
}

func (s *DashboardService) getExecutorStats(stats *models.DashboardStats, userID uuid.UUID) (*models.DashboardStats, error) {
	// 1. Counts by status
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

	// 2. Overdue count (status in ('new', 'in_progress') AND deadline < NOW())
	// OR status is 'completed' AND completed_at::date > deadline
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

	// 2.1 Finished (total) and Finished Late
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

	// 3. Expiring assignments (deadline within next 3 days)
	// Only active assignments
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
		a.FillIDStr()
		stats.ExpiringAssignments = append(stats.ExpiringAssignments, a)
	}

	return stats, nil
}

func (s *DashboardService) getClerkStats(stats *models.DashboardStats, startDate, endDate time.Time) (*models.DashboardStats, error) {
	// 1. Doc counts for period
	err := s.db.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM incoming_documents WHERE created_at BETWEEN $1 AND $2),
			(SELECT COUNT(*) FROM outgoing_documents WHERE created_at BETWEEN $1 AND $2)
	`, startDate, endDate).Scan(&stats.IncomingCount, &stats.OutgoingCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get doc counts: %w", err)
	}

	// 2. All overdue count
	// Strict interpretation: Assignments that are overdue NOW.
	// Period interpretation: Assignments with deadline IN PERIOD that are overdue?
	// The user asked "statistics for documents and assignments [for a period]".
	// For overdue, usually you want to know what is overdue *right now*, regardless of when it was created.
	// However, if we must apply the period, "Overdue projects started/deadlined in this period" makes sense.
	// Let's stick to "Deadline >= startDate" for consistency if we want "Overdue assignments OF THIS PERIOD".
	// But commonly "Overdue" is a current state.
	// The request says "for statistics ... make choice of period".
	// Let's assume the user wants to see stats relevant to that period.
	// For "Overdue", it might mean "Assignments with deadline in this period that are overdue".
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

	// 3. Finished (all time in period) and Finished Late - NEW
	// Fallback to updated_at if completed_at is NULL (for old data)
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

	// 3. All expiring assignments (global) - Increased interval to 7 days for clerks
	// Expiring is always "Future", so period doesn't quite apply, or it applies to "Active assignments in this period"?
	// "Expiring" list is usually "What to look at NOW". Unlikely to need period filter here.
	// We will leave expiring list as "Next 7 days from NOW".
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
		a.FillIDStr()
		stats.ExpiringAssignments = append(stats.ExpiringAssignments, a)
	}

	return stats, nil
}

func (s *DashboardService) getAdminStats(stats *models.DashboardStats) (*models.DashboardStats, error) {
	// 1. User count
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.UserCount)
	if err != nil {
		return nil, err
	}

	// 2. Total docs
	var inc, out int
	err = s.db.QueryRow("SELECT (SELECT COUNT(*) FROM incoming_documents), (SELECT COUNT(*) FROM outgoing_documents)").Scan(&inc, &out)
	if err != nil {
		return nil, err
	}
	stats.TotalDocuments = inc + out

	// 3. DB Size (Postgres specific) - handled gracefully
	err = s.db.QueryRow("SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&stats.DBSize)
	if err != nil {
		stats.DBSize = "N/A"
	}

	return stats, nil
}

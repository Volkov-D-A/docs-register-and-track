package services

import (
	"context"
	"database/sql"
	"fmt"

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

func (s *DashboardService) GetStats() (*models.DashboardStats, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	role := "executor"
	if s.auth.HasRole("admin") {
		role = "admin"
	} else if s.auth.HasRole("clerk") {
		role = "clerk"
	}

	// Initialize with empty slice to avoid null in JSON
	stats := &models.DashboardStats{
		Role:                role,
		ExpiringAssignments: []models.Assignment{},
	}

	if role == "admin" {
		return s.getAdminStats(stats)
	} else if role == "clerk" {
		return s.getClerkStats(stats)
	} else {
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
		FROM assignments 
		WHERE executor_id = $1
	`, userID).Scan(&stats.MyAssignmentsNew, &stats.MyAssignmentsInProgress)
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	// 2. Overdue count (status in ('new', 'in_progress') AND deadline < NOW())
	err = s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM assignments 
		WHERE executor_id = $1 
		  AND status IN ('new', 'in_progress') 
		  AND deadline < CURRENT_DATE
	`, userID).Scan(&stats.MyAssignmentsOverdue)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue count: %w", err)
	}

	// 3. Expiring assignments (deadline within next 3 days)
	rows, err := s.db.Query(`
		SELECT 
			a.id, a.content, a.deadline, a.status,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number
		FROM assignments a
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
		WHERE a.executor_id = $1 
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

		if err := rows.Scan(&a.ID, &a.Content, &deadline, &a.Status, &docNumber); err != nil {
			return nil, err
		}
		if deadline.Valid {
			a.Deadline = &deadline.Time
		}
		if docNumber.Valid {
			a.DocumentNumber = docNumber.String
		}
		a.FillIDStr()
		stats.ExpiringAssignments = append(stats.ExpiringAssignments, a)
	}

	return stats, nil
}

func (s *DashboardService) getClerkStats(stats *models.DashboardStats) (*models.DashboardStats, error) {
	// 1. Doc counts for current month - Fixed date_trunc
	err := s.db.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM incoming_documents WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE)),
			(SELECT COUNT(*) FROM outgoing_documents WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE))
	`).Scan(&stats.IncomingCountMonth, &stats.OutgoingCountMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to get doc counts: %w", err)
	}

	// 2. All overdue count
	err = s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM assignments 
		WHERE status IN ('new', 'in_progress') 
		  AND deadline < CURRENT_DATE
	`).Scan(&stats.AllAssignmentsOverdue)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue count: %w", err)
	}

	// 3. All expiring assignments (global) - Increased interval to 7 days for clerks
	rows, err := s.db.Query(`
		SELECT 
			a.id, a.content, a.deadline, a.status,
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

		if err := rows.Scan(&a.ID, &a.Content, &deadline, &a.Status, &executorName, &docNumber); err != nil {
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

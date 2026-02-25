package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/database"
	"docflow/internal/models"
)

// DashboardRepository — репозиторий для запросов дашборда.
type DashboardRepository struct {
	db *database.DB
}

func NewDashboardRepository(db *database.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// --- Executor stats ---

// GetExecutorStatusCounts возвращает количество поручений по статусам new и in_progress для исполнителя.
func (r *DashboardRepository) GetExecutorStatusCounts(userID uuid.UUID) (newCount, inProgressCount int, err error) {
	err = r.db.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'new'),
			COUNT(*) FILTER (WHERE status = 'in_progress')
		FROM assignments a
		WHERE executor_id = $1
		OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $1)
	`, userID).Scan(&newCount, &inProgressCount)
	if err != nil {
		err = fmt.Errorf("failed to get executor status counts: %w", err)
	}
	return
}

// GetExecutorOverdueCount возвращает количество просроченных поручений для исполнителя.
func (r *DashboardRepository) GetExecutorOverdueCount(userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) 
		FROM assignments a
		WHERE (executor_id = $1 OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $1))
		  AND (
		      (status IN ('new', 'in_progress') AND deadline < CURRENT_DATE)
		      OR
		      (status = 'completed' AND completed_at::date > deadline)
		  )
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get executor overdue count: %w", err)
	}
	return count, nil
}

// GetExecutorFinishedCounts возвращает количество завершённых поручений (всего и с опозданием) для исполнителя.
func (r *DashboardRepository) GetExecutorFinishedCounts(userID uuid.UUID) (finished, finishedLate int, err error) {
	err = r.db.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'finished'),
			COUNT(*) FILTER (WHERE status = 'finished' AND completed_at::date > deadline)
		FROM assignments a
		WHERE (executor_id = $1 OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $1))
	`, userID).Scan(&finished, &finishedLate)
	if err != nil {
		err = fmt.Errorf("failed to get executor finished counts: %w", err)
	}
	return
}

// GetExpiringAssignments возвращает поручения, срок которых истекает в ближайшие N дней.
// Если userID не nil — фильтрует по исполнителю, иначе — все поручения.
func (r *DashboardRepository) GetExpiringAssignments(userID *uuid.UUID, days int) ([]models.Assignment, error) {
	var rows *sql.Rows
	var err error

	if userID != nil {
		rows, err = r.db.Query(`
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
			  AND a.deadline BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '1 day' * $2)
			ORDER BY a.deadline ASC
		`, *userID, days)
	} else {
		rows, err = r.db.Query(`
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
			  AND a.deadline BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '1 day' * $1)
			ORDER BY a.deadline ASC
		`, days)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring assignments: %w", err)
	}
	defer rows.Close()

	assignments := make([]models.Assignment, 0)
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

		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return assignments, nil
}

// --- Clerk stats ---

// GetDocCountsByPeriod возвращает количество входящих и исходящих документов за период.
func (r *DashboardRepository) GetDocCountsByPeriod(startDate, endDate time.Time) (incoming, outgoing int, err error) {
	err = r.db.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM incoming_documents WHERE created_at BETWEEN $1 AND $2),
			(SELECT COUNT(*) FROM outgoing_documents WHERE created_at BETWEEN $1 AND $2)
	`, startDate, endDate).Scan(&incoming, &outgoing)
	if err != nil {
		err = fmt.Errorf("failed to get doc counts by period: %w", err)
	}
	return
}

// GetOverdueCountByPeriod возвращает количество просроченных поручений за период.
func (r *DashboardRepository) GetOverdueCountByPeriod(startDate, endDate time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) 
		FROM assignments 
		WHERE deadline BETWEEN $1 AND $2 
		  AND (
		      (status IN ('new', 'in_progress') AND deadline < CURRENT_DATE)
		      OR 
		      (status = 'completed' AND completed_at::date > deadline)
		  )
	`, startDate, endDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get overdue count by period: %w", err)
	}
	return count, nil
}

// GetFinishedCountsByPeriod возвращает количество завершённых поручений за период (всего и с опозданием).
func (r *DashboardRepository) GetFinishedCountsByPeriod(startDate, endDate time.Time) (finished, finishedLate int, err error) {
	err = r.db.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'finished'),
			COUNT(*) FILTER (WHERE status = 'finished' AND COALESCE(completed_at, updated_at)::date > deadline)
		FROM assignments
		WHERE status = 'finished' AND COALESCE(completed_at, updated_at) BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&finished, &finishedLate)
	if err != nil {
		err = fmt.Errorf("failed to get finished counts by period: %w", err)
	}
	return
}

// --- Admin stats ---

// GetAdminUserCount возвращает общее количество пользователей.
func (r *DashboardRepository) GetAdminUserCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}
	return count, nil
}

// GetAdminDocCounts возвращает общее количество входящих и исходящих документов.
func (r *DashboardRepository) GetAdminDocCounts() (incoming, outgoing int, err error) {
	err = r.db.QueryRow("SELECT (SELECT COUNT(*) FROM incoming_documents), (SELECT COUNT(*) FROM outgoing_documents)").Scan(&incoming, &outgoing)
	if err != nil {
		err = fmt.Errorf("failed to get admin doc counts: %w", err)
	}
	return
}

// GetDBSize возвращает размер базы данных в человекочитаемом формате.
func (r *DashboardRepository) GetDBSize() string {
	var size string
	err := r.db.QueryRow("SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&size)
	if err != nil {
		return "N/A"
	}
	return size
}

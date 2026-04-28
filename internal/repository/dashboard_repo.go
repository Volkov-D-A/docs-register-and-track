package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DashboardRepository — репозиторий для запросов дашборда.
type DashboardRepository struct {
	db *database.DB
}

// NewDashboardRepository создает новый экземпляр DashboardRepository.
func NewDashboardRepository(db *database.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
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
				a.document_id, d.kind,
				u.full_name as executor_name,
				d.registration_number as doc_number
			FROM assignments a
			JOIN documents d ON d.id = a.document_id
			LEFT JOIN users u ON a.executor_id = u.id
			WHERE (a.executor_id = $1 OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $1))
			  AND a.status IN ('new', 'in_progress')
			  AND a.deadline BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '1 day' * $2)
			ORDER BY a.deadline ASC
		`, *userID, days)
	} else {
		rows, err = r.db.Query(`
			SELECT 
				a.id, a.content, a.deadline, a.status,
				a.document_id, d.kind,
				u.full_name as executor_name,
				d.registration_number as doc_number
			FROM assignments a
			JOIN documents d ON d.id = a.document_id
			LEFT JOIN users u ON a.executor_id = u.id
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

		if err := rows.Scan(&a.ID, &a.Content, &deadline, &a.Status, &a.DocumentID, &a.DocumentKind, &executorName, &docNumber); err != nil {
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

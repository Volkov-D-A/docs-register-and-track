package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"

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

// GetExpiringAssignments возвращает поручения, срок которых истекает в ближайшие N дней,
// ограниченные серверным scope доступа к документам.
func (r *DashboardRepository) GetExpiringAssignments(filter models.DashboardAssignmentFilter) ([]models.Assignment, error) {
	where := []string{
		"a.status IN ('new', 'in_progress')",
		"a.deadline BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '1 day' * $1)",
	}
	args := []interface{}{filter.Days}
	argIdx := 2
	if len(filter.AllowedDocumentKinds) > 0 || len(filter.AccessibleByUserIDs) > 0 {
		accessClauses := make([]string, 0, 2)
		if len(filter.AllowedDocumentKinds) > 0 {
			accessClauses = append(accessClauses, fmt.Sprintf("d.kind = ANY($%d)", argIdx))
			args = append(args, pq.Array(filter.AllowedDocumentKinds))
			argIdx++
		}
		if len(filter.AccessibleByUserIDs) > 0 {
			accessClauses = append(accessClauses, fmt.Sprintf("(a.executor_id = ANY($%d::uuid[]) OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = ANY($%d::uuid[])))", argIdx, argIdx))
			args = append(args, pq.Array(filter.AccessibleByUserIDs))
		}
		where = append(where, "("+strings.Join(accessClauses, " OR ")+")")
	}

	rows, err := r.db.Query(`
			SELECT 
				a.id, a.content, a.deadline, a.status,
				a.document_id, d.kind,
				u.full_name as executor_name,
				d.registration_number as doc_number
			FROM assignments a
			JOIN documents d ON d.id = a.document_id
			LEFT JOIN users u ON a.executor_id = u.id
			WHERE `+strings.Join(where, " AND ")+`
			ORDER BY a.deadline ASC
		`, args...)
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

package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

// WorkspaceRepository computes counts and bounded previews under server-resolved scopes.
type WorkspaceRepository struct{ db *database.DB }

func NewWorkspaceRepository(db *database.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

func workspaceWhere(query models.WorkspaceQuery) (string, []interface{}, error) {
	if query.Mode != models.WorkspaceModeExecution && query.Mode != models.WorkspaceModeControl {
		return "", nil, fmt.Errorf("invalid workspace mode %q", query.Mode)
	}
	where := []string{}
	args := []interface{}{}
	argIdx := 1
	if query.Mode == models.WorkspaceModeExecution {
		if len(query.SubjectIDs) == 0 {
			return "1=0", args, nil
		}
		where = append(where, fmt.Sprintf(`((a.type = 'execution' AND (a.executor_id = ANY($%d::uuid[]) OR EXISTS (
            SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = ANY($%d::uuid[]))))
            OR (a.type = 'acknowledgment' AND EXISTS (SELECT 1 FROM assignment_recipients ar
            WHERE ar.assignment_id = a.id AND ar.user_id = ANY($%d::uuid[]) AND ar.confirmed_at IS NULL)))`, argIdx, argIdx, argIdx))
		args = append(args, pq.Array(query.SubjectIDs))
	} else {
		if query.Kind == "" {
			return "", nil, fmt.Errorf("workspace control query requires document kind")
		}
		where = append(where, fmt.Sprintf("d.kind = $%d", argIdx))
		args = append(args, string(query.Kind))
		argIdx++
		applyDocumentListAccess(&where, &args, &argIdx, query.ReadScope)
	}
	return strings.Join(where, " AND "), args, nil
}

func (r *WorkspaceRepository) AssignmentSummary(query models.WorkspaceQuery) (models.WorkspaceAssignmentCounts, []models.WorkspaceAssignment, error) {
	var counts models.WorkspaceAssignmentCounts
	accessWhere, args, err := workspaceWhere(query)
	if err != nil {
		return counts, nil, err
	}
	where := []string{
		"(a.series_id IS NULL OR s.current_assignment_id = a.id)",
		"a.status IN ('new', 'in_progress', 'returned', 'completed')",
		accessWhere,
	}
	if query.Mode == models.WorkspaceModeExecution {
		where = append(where, "a.status != 'completed'")
	}
	from := ` FROM assignments a JOIN documents d ON d.id = a.document_id
		LEFT JOIN assignment_series s ON s.id = a.series_id
		WHERE ` + strings.Join(where, " AND ")
	countSQL := `SELECT
		COUNT(*) FILTER (WHERE a.status = 'new'),
		COUNT(*) FILTER (WHERE a.status = 'in_progress'),
		COUNT(*) FILTER (WHERE a.status = 'returned'),
		COUNT(*) FILTER (WHERE a.status != 'completed' AND a.deadline::date < CURRENT_DATE),
		COUNT(*) FILTER (WHERE a.deadline::date BETWEEN CURRENT_DATE AND CURRENT_DATE + 3),
		COUNT(*) FILTER (WHERE a.status = 'completed')` + from
	if err := r.db.QueryRow(countSQL, args...).Scan(&counts.New, &counts.InProgress, &counts.Returned, &counts.Overdue, &counts.DueSoon, &counts.AwaitingAcceptance); err != nil {
		return counts, nil, err
	}
	limit := query.Limit
	if limit < 1 || limit > 20 {
		limit = 5
	}
	rows, err := r.db.Query(`SELECT a.id, a.document_id, d.kind, d.registration_number, d.registration_date,
		a.content, a.deadline, a.status, a.type, d.content, a.created_at`+from+`
		ORDER BY CASE WHEN a.status = 'completed' THEN 0 WHEN a.deadline::date < CURRENT_DATE THEN 1 ELSE 2 END,
		a.deadline ASC NULLS LAST, a.created_at DESC, a.id LIMIT `+fmt.Sprint(limit), args...)
	if err != nil {
		return counts, nil, err
	}
	defer rows.Close()
	items := []models.WorkspaceAssignment{}
	for rows.Next() {
		var item models.WorkspaceAssignment
		var deadline sql.NullTime
		if err := rows.Scan(&item.ID, &item.DocumentID, &item.DocumentKind, &item.DocumentNumber,
			&item.DocumentDate, &item.Content, &deadline, &item.Status, &item.Type, &item.DocumentContent, &item.CreatedAt); err != nil {
			return counts, nil, err
		}
		if deadline.Valid {
			item.Deadline = &deadline.Time
		}
		items = append(items, item)
	}
	return counts, items, rows.Err()
}

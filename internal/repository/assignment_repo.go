package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"docflow/internal/database"
	"docflow/internal/models"
)

type AssignmentRepository struct {
	db *database.DB
}

func NewAssignmentRepository(db *database.DB) *AssignmentRepository {
	return &AssignmentRepository{db: db}
}

func (r *AssignmentRepository) Create(
	documentID uuid.UUID,
	documentType string,
	executorID uuid.UUID,
	content string,
	deadline *time.Time,
) (*models.Assignment, error) {
	var id uuid.UUID
	var createdAt, updatedAt time.Time
	var status = "new"

	query := `
		INSERT INTO assignments (
			document_id, document_type, executor_id,
			content, deadline, status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		documentID, documentType, executorID,
		content, deadline, status,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	return r.GetByID(id)
}

func (r *AssignmentRepository) Update(
	id uuid.UUID,
	executorID uuid.UUID,
	content string,
	deadline *time.Time,
	status, report string,
) (*models.Assignment, error) {
	query := `
		UPDATE assignments
		SET executor_id = $1, content = $2, deadline = $3,
		    status = $4, report = $5, updated_at = NOW()
		WHERE id = $6
	`

	_, err := r.db.Exec(query, executorID, content, deadline, status, report, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update assignment: %w", err)
	}

	return r.GetByID(id)
}

func (r *AssignmentRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM assignments WHERE id = $1", id)
	return err
}

func (r *AssignmentRepository) GetByID(id uuid.UUID) (*models.Assignment, error) {
	query := `
		SELECT
			a.id, a.document_id, a.document_type,
			a.executor_id, u_executor.full_name,
			a.content, a.deadline, a.status, a.report,
			a.created_at, a.updated_at,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number,
			COALESCE(inc.subject, out.subject) as doc_subject
		FROM assignments a
		LEFT JOIN users u_executor ON a.executor_id = u_executor.id
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
		WHERE a.id = $1
	`

	var a models.Assignment
	var deadline sql.NullTime
	var report sql.NullString
	var docNumber sql.NullString
	var docSubject sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&a.ID, &a.DocumentID, &a.DocumentType,
		&a.ExecutorID, &a.ExecutorName,
		&a.Content, &deadline, &a.Status, &report,
		&a.CreatedAt, &a.UpdatedAt,
		&docNumber, &docSubject,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	if deadline.Valid {
		a.Deadline = &deadline.Time
	}
	if report.Valid {
		a.Report = report.String
	}
	if docNumber.Valid {
		a.DocumentNumber = docNumber.String
	}
	if docSubject.Valid {
		a.DocumentSubject = docSubject.String
	}

	a.FillIDStr()
	return &a, nil
}

func (r *AssignmentRepository) GetList(filter models.AssignmentFilter) (*models.PagedResult, error) {
	query := `
		SELECT
			a.id, a.document_id, a.document_type,
			a.executor_id, u_executor.full_name,
			a.content, a.deadline, a.status, a.report,
			a.created_at, a.updated_at,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number,
			COALESCE(inc.subject, out.subject) as doc_subject
		FROM assignments a
		LEFT JOIN users u_executor ON a.executor_id = u_executor.id
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
	`

	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.DocumentID != "" {
		where = append(where, fmt.Sprintf("a.document_id = $%d", argIdx))
		args = append(args, filter.DocumentID)
		argIdx++
	}
	if filter.ExecutorID != "" {
		where = append(where, fmt.Sprintf("a.executor_id = $%d", argIdx))
		args = append(args, filter.ExecutorID)
		argIdx++
	}
	if filter.Status != "" {
		where = append(where, fmt.Sprintf("a.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("a.created_at >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("a.created_at <= $%d", argIdx))
		args = append(args, filter.DateTo+" 23:59:59")
		argIdx++
	}

	if filter.Search != "" {
		search := "%" + strings.ToLower(filter.Search) + "%"
		where = append(where, fmt.Sprintf("(LOWER(a.content) LIKE $%d OR LOWER(inc.incoming_number) LIKE $%d OR LOWER(inc.subject) LIKE $%d OR LOWER(out.outgoing_number) LIKE $%d OR LOWER(out.subject) LIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, search, search, search, search, search)
		argIdx++
	}

	query += " WHERE " + strings.Join(where, " AND ")

	// Count query
	countQuery := "SELECT COUNT(*) FROM assignments a WHERE " + strings.Join(where, " AND ")
	var totalCount int
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count assignments: %w", err)
	}

	// Pagination
	query += fmt.Sprintf(" ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}
	defer rows.Close()

	var items []models.Assignment
	for rows.Next() {
		var a models.Assignment
		var deadline sql.NullTime
		var report sql.NullString
		var docNumber sql.NullString
		var docSubject sql.NullString

		if err := rows.Scan(
			&a.ID, &a.DocumentID, &a.DocumentType,
			&a.ExecutorID, &a.ExecutorName,
			&a.Content, &deadline, &a.Status, &report,
			&a.CreatedAt, &a.UpdatedAt,
			&docNumber, &docSubject,
		); err != nil {
			return nil, err
		}

		if deadline.Valid {
			a.Deadline = &deadline.Time
		}
		if report.Valid {
			a.Report = report.String
		}
		if docNumber.Valid {
			a.DocumentNumber = docNumber.String
		}
		if docSubject.Valid {
			a.DocumentSubject = docSubject.String
		}
		a.FillIDStr()
		items = append(items, a)
	}

	return &models.PagedResult{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

// GetCountByStatus — вспомогательный метод для дашборда
func (r *AssignmentRepository) GetCountByStatus(status string, executorID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM assignments WHERE status = $1 AND executor_id = $2",
		status, executorID,
	).Scan(&count)
	return count, err
}

package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

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
	coExecutorIDs []string,
) (*models.Assignment, error) {
	var id uuid.UUID
	var createdAt, updatedAt time.Time
	var status = "new"

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO assignments (
			document_id, document_type, executor_id,
			content, deadline, status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(
		query,
		documentID, documentType, executorID,
		content, deadline, status,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	// Insert co-executors
	if len(coExecutorIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO assignment_co_executors (assignment_id, user_id) VALUES ($1, $2)")
		if err != nil {
			return nil, fmt.Errorf("failed to prepare co-executors statement: %w", err)
		}
		defer stmt.Close()

		for _, coExecID := range coExecutorIDs {
			uid, err := uuid.Parse(coExecID)
			if err != nil {
				return nil, fmt.Errorf("invalid co-executor ID %s: %w", coExecID, err)
			}
			if _, err := stmt.Exec(id, uid); err != nil {
				return nil, fmt.Errorf("failed to insert co-executor %s: %w", coExecID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(id)
}

func (r *AssignmentRepository) Update(
	id uuid.UUID,
	executorID uuid.UUID,
	content string,
	deadline *time.Time,
	status, report string,
	completedAt *time.Time,
	coExecutorIDs []string,
) (*models.Assignment, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE assignments
		SET executor_id = $1, content = $2, deadline = $3,
		    status = $4, report = $5, completed_at = $6, updated_at = NOW()
		WHERE id = $7
	`

	_, err = tx.Exec(query, executorID, content, deadline, status, report, completedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update assignment: %w", err)
	}

	// Update co-executors
	// 1. Delete existing
	if _, err := tx.Exec("DELETE FROM assignment_co_executors WHERE assignment_id = $1", id); err != nil {
		return nil, fmt.Errorf("failed to delete old co-executors: %w", err)
	}

	// 2. Insert new
	if len(coExecutorIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO assignment_co_executors (assignment_id, user_id) VALUES ($1, $2)")
		if err != nil {
			return nil, fmt.Errorf("failed to prepare co-executors statement: %w", err)
		}
		defer stmt.Close()

		for _, coExecID := range coExecutorIDs {
			uid, err := uuid.Parse(coExecID)
			if err != nil {
				return nil, fmt.Errorf("invalid co-executor ID %s: %w", coExecID, err)
			}
			if _, err := stmt.Exec(id, uid); err != nil {
				return nil, fmt.Errorf("failed to insert co-executor %s: %w", coExecID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
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
			a.content, a.deadline, a.status, a.report, a.completed_at,
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
	var completedAt sql.NullTime
	var report sql.NullString
	var docNumber sql.NullString
	var docSubject sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&a.ID, &a.DocumentID, &a.DocumentType,
		&a.ExecutorID, &a.ExecutorName,
		&a.Content, &deadline, &a.Status, &report, &completedAt,
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
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
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

	// Fetch co-executors
	coExecQuery := `
		SELECT u.id, u.login, u.full_name
		FROM assignment_co_executors ce
		JOIN users u ON ce.user_id = u.id
		WHERE ce.assignment_id = $1
	`
	ceRows, err := r.db.Query(coExecQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get co-executors: %w", err)
	}
	defer ceRows.Close()

	var coExecutors []models.User
	var coExecutorIDs []string

	for ceRows.Next() {
		var u models.User
		if err := ceRows.Scan(&u.ID, &u.Login, &u.FullName); err != nil {
			return nil, err
		}
		u.FillIDStr()
		coExecutors = append(coExecutors, u)
		coExecutorIDs = append(coExecutorIDs, u.IDStr)
	}
	a.CoExecutors = coExecutors
	a.CoExecutorIDs = coExecutorIDs

	return &a, nil
}

func (r *AssignmentRepository) GetList(filter models.AssignmentFilter) (*models.PagedResult[models.Assignment], error) {
	query := `
		SELECT
			a.id, a.document_id, a.document_type,
			a.executor_id, u_executor.full_name,
			a.content, a.deadline, a.status, a.report, a.completed_at,
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
		// Filter by main executor OR co-executor
		where = append(where, fmt.Sprintf("(a.executor_id = $%d OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $%d))", argIdx, argIdx))
		args = append(args, filter.ExecutorID)
		argIdx++
	}

	if filter.OverdueOnly {
		// Overdue: deadline < current_date AND status not in (completed, finished, cancelled)
		// OR status is completed but completed_at > deadline
		// We use deadline < CURRENT_DATE because if deadline is today, it's not overdue yet until tomorrow?
		// Usually "overdue" means deadline < today.
		// However, some definitions allow today. Let's assume deadline < NOW() or CURRENT_DATE.
		// Let's use strict CURRENT_DATE comparison for "deadline < today".
		where = append(where, "(a.deadline < CURRENT_DATE AND (a.status NOT IN ('completed', 'finished', 'cancelled') OR (a.status = 'completed' AND a.completed_at::date > a.deadline)))")
	}

	if filter.Status != "" {
		// If ShowFinished is false, we must strictly forbid "finished" status
		// even if the user explicitly requested it (though frontend should prevent this).
		if !filter.ShowFinished && filter.Status == "finished" {
			// Return empty result efficiently
			return &models.PagedResult[models.Assignment]{Items: []models.Assignment{}, TotalCount: 0, Page: filter.Page, PageSize: filter.PageSize}, nil
		}

		where = append(where, fmt.Sprintf("a.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	} else {
		// If status is not specified, hide 'finished' unless ShowFinished is true
		if !filter.ShowFinished {
			where = append(where, fmt.Sprintf("a.status != $%d", argIdx))
			args = append(args, "finished")
			argIdx++
		}
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("a.deadline >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("a.deadline <= $%d", argIdx))
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
	countQuery := "SELECT COUNT(*) FROM assignments a LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming' LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing' WHERE " + strings.Join(where, " AND ")
	var totalCount int
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count assignments: %w", err)
	}

	// Pagination defaults
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
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
	var assignmentIDs []uuid.UUID
	assignmentIndex := map[uuid.UUID]int{} // assignment ID -> index in items

	for rows.Next() {
		var a models.Assignment
		var deadline sql.NullTime
		var completedAt sql.NullTime
		var report sql.NullString
		var docNumber sql.NullString
		var docSubject sql.NullString

		if err := rows.Scan(
			&a.ID, &a.DocumentID, &a.DocumentType,
			&a.ExecutorID, &a.ExecutorName,
			&a.Content, &deadline, &a.Status, &report, &completedAt,
			&a.CreatedAt, &a.UpdatedAt,
			&docNumber, &docSubject,
		); err != nil {
			return nil, err
		}

		if deadline.Valid {
			a.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
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

		assignmentIndex[a.ID] = len(items)
		assignmentIDs = append(assignmentIDs, a.ID)
		items = append(items, a)
	}

	// Batch-fetch co-executors for all assignments in one query instead of N+1
	if len(assignmentIDs) > 0 {
		coExecQuery := `
			SELECT ce.assignment_id, u.id, u.login, u.full_name
			FROM assignment_co_executors ce
			JOIN users u ON ce.user_id = u.id
			WHERE ce.assignment_id = ANY($1)
		`
		ceRows, err := r.db.Query(coExecQuery, pq.Array(assignmentIDs))
		if err != nil {
			return nil, fmt.Errorf("failed to get co-executors: %w", err)
		}
		defer ceRows.Close()

		for ceRows.Next() {
			var assignmentID uuid.UUID
			var u models.User
			if err := ceRows.Scan(&assignmentID, &u.ID, &u.Login, &u.FullName); err != nil {
				return nil, fmt.Errorf("failed to scan co-executor: %w", err)
			}
			u.FillIDStr()

			if idx, ok := assignmentIndex[assignmentID]; ok {
				items[idx].CoExecutors = append(items[idx].CoExecutors, u)
				items[idx].CoExecutorIDs = append(items[idx].CoExecutorIDs, u.IDStr)
			}
		}
	}

	return &models.PagedResult[models.Assignment]{
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

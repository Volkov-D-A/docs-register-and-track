package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

// AssignmentRepository предоставляет методы для работы с поручениями в БД.
type AssignmentRepository struct {
	db     *database.DB
	outbox *OutboxRepository
}

func (r *AssignmentRepository) SetOutbox(outbox *OutboxRepository) { r.outbox = outbox }

// CreateWithOutbox persists the assignment and all supplied effects in one
// transaction. It is used by production services that require no post-commit gap.
func (r *AssignmentRepository) CreateWithOutbox(id, documentID, executorID uuid.UUID, content string, deadline *time.Time, coExecutorIDs []string, effects []models.OutboxEvent) (*models.Assignment, error) {
	if r.outbox == nil {
		return nil, ErrOutboxNotConfigured
	}
	var createdAt, updatedAt time.Time
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	err = tx.QueryRow(`INSERT INTO assignments (id, document_id, executor_id, content, deadline, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at, updated_at`, id, documentID, executorID, content, deadline, "new").Scan(&createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}
	for _, coExecID := range coExecutorIDs {
		uid, err := uuid.Parse(coExecID)
		if err != nil {
			return nil, fmt.Errorf("invalid co-executor ID %s: %w", coExecID, err)
		}
		if _, err = tx.Exec("INSERT INTO assignment_co_executors (assignment_id, user_id) VALUES ($1, $2)", id, uid); err != nil {
			return nil, err
		}
	}
	for _, effect := range effects {
		if err := r.outbox.EnqueueTx(tx, effect); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return r.GetByID(id)
}

// NewAssignmentRepository создает новый экземпляр AssignmentRepository.
func NewAssignmentRepository(db *database.DB) *AssignmentRepository {
	return &AssignmentRepository{db: db}
}

func (r *AssignmentRepository) UpdateWithOutbox(id, executorID uuid.UUID, content string, deadline *time.Time, status, report string, completedAt *time.Time, coExecutorIDs []string, effects []models.OutboxEvent) (*models.Assignment, error) {
	if r.outbox == nil {
		return nil, ErrOutboxNotConfigured
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE assignments SET executor_id=$1, content=$2, deadline=$3, status=$4, report=$5, completed_at=$6, updated_at=NOW() WHERE id=$7`, executorID, content, deadline, status, report, completedAt, id); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("DELETE FROM assignment_co_executors WHERE assignment_id = $1", id); err != nil {
		return nil, err
	}
	for _, value := range coExecutorIDs {
		uid, parseErr := uuid.Parse(value)
		if parseErr != nil {
			return nil, parseErr
		}
		if _, err = tx.Exec("INSERT INTO assignment_co_executors (assignment_id, user_id) VALUES ($1, $2)", id, uid); err != nil {
			return nil, err
		}
	}
	for _, effect := range effects {
		if err = r.outbox.EnqueueTx(tx, effect); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

// UpdateDetailsWithOutbox updates only editable assignment fields and rejects a
// stale editor snapshot. In particular, it cannot overwrite a concurrent
// status transition or modify an iteration after it has been accepted.
func (r *AssignmentRepository) UpdateDetailsWithOutbox(id, executorID uuid.UUID, content string, deadline *time.Time, coExecutorIDs []string, expectedUpdatedAt time.Time, effects []models.OutboxEvent) (*models.Assignment, error) {
	if r.outbox == nil {
		return nil, ErrOutboxNotConfigured
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE assignments
		SET executor_id=$1, content=$2, deadline=$3, updated_at=NOW()
		WHERE id=$4 AND updated_at=$5 AND status <> 'finished'`, executorID, content, deadline, id, expectedUpdatedAt)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, models.NewConflict("поручение было изменено; обновите данные и повторите")
	}
	if _, err = tx.Exec("DELETE FROM assignment_co_executors WHERE assignment_id = $1", id); err != nil {
		return nil, err
	}
	for _, value := range coExecutorIDs {
		uid, parseErr := uuid.Parse(value)
		if parseErr != nil {
			return nil, parseErr
		}
		if _, err = tx.Exec("INSERT INTO assignment_co_executors (assignment_id, user_id) VALUES ($1, $2)", id, uid); err != nil {
			return nil, err
		}
	}
	for _, effect := range effects {
		if err = r.outbox.EnqueueTx(tx, effect); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *AssignmentRepository) DeleteWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error {
	if r.outbox == nil {
		return ErrOutboxNotConfigured
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("DELETE FROM assignments WHERE id = $1", id); err != nil {
		return err
	}
	for _, effect := range effects {
		if err = r.outbox.EnqueueTx(tx, effect); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetByID возвращает поручение по его ID.
func (r *AssignmentRepository) GetByID(id uuid.UUID) (*models.Assignment, error) {
	query := `
		SELECT
			a.id, a.document_id, d.kind,
			COALESCE(a.executor_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(u_executor.full_name, ''),
			CASE WHEN a.type = 'execution' THEN a.content ELSE '' END, a.deadline, a.status, a.report, a.completed_at,
			a.series_id, a.iteration_number, a.planned_deadline,
			COALESCE(s.current_assignment_id = a.id, FALSE) AS is_series_current,
			a.created_at, a.updated_at,
			d.registration_number as doc_number,
			d.content as doc_subject, a.type, a.creator_id, COALESCE(u_creator.full_name, '')
		FROM assignments a
		JOIN documents d ON d.id = a.document_id
		LEFT JOIN users u_executor ON a.executor_id = u_executor.id
		LEFT JOIN users u_creator ON a.creator_id = u_creator.id
		LEFT JOIN assignment_series s ON s.id = a.series_id
		WHERE a.id = $1
	`

	var a models.Assignment
	var deadline sql.NullTime
	var completedAt sql.NullTime
	var report sql.NullString
	var docNumber sql.NullString
	var docSubject sql.NullString
	var seriesID uuid.NullUUID
	var iterationNumber sql.NullInt64
	var plannedDeadline sql.NullTime
	var creatorID uuid.NullUUID

	err := r.db.QueryRow(query, id).Scan(
		&a.ID, &a.DocumentID, &a.DocumentKind,
		&a.ExecutorID, &a.ExecutorName,
		&a.Content, &deadline, &a.Status, &report, &completedAt,
		&seriesID, &iterationNumber, &plannedDeadline, &a.IsSeriesCurrent,
		&a.CreatedAt, &a.UpdatedAt,
		&docNumber, &docSubject, &a.Type, &creatorID, &a.CreatorName,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Не найдено
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	if creatorID.Valid {
		a.CreatorID = creatorID.UUID
	}
	if deadline.Valid {
		a.Deadline = &deadline.Time
	}
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
	}
	if seriesID.Valid {
		a.SeriesID = &seriesID.UUID
	}
	if iterationNumber.Valid {
		a.IterationNumber = int(iterationNumber.Int64)
	}
	if plannedDeadline.Valid {
		a.PlannedDeadline = &plannedDeadline.Time
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

	// Получение соисполнителей
	coExecQuery := `
		SELECT u.id, u.login, u.last_name, u.first_name, u.patronymic, u.no_patronymic
		FROM assignment_co_executors ce
		JOIN users u ON ce.user_id = u.id
		WHERE ce.assignment_id = $1
	`
	ceRows, err := r.db.Query(coExecQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get co-executors: %w", err)
	}
	defer ceRows.Close()

	coExecutors := make([]models.User, 0)
	coExecutorIDs := make([]string, 0)

	for ceRows.Next() {
		var u models.User
		if err := ceRows.Scan(&u.ID, &u.Login, &u.LastName, &u.FirstName, &u.Patronymic, &u.NoPatronymic); err != nil {
			return nil, err
		}

		coExecutors = append(coExecutors, u)
		coExecutorIDs = append(coExecutorIDs, u.ID.String())
	}
	if err := ceRows.Err(); err != nil {
		return nil, err
	}
	a.CoExecutors = coExecutors
	a.CoExecutorIDs = coExecutorIDs

	if a.Type == models.AssignmentTypeAcknowledgment {
		recipients, err := r.GetRecipientsByAssignmentIDs([]uuid.UUID{a.ID})
		if err != nil {
			return nil, err
		}
		a.Users = recipients[a.ID]
	}
	return &a, nil
}

// GetList возвращает список поручений с учетом фильтрации и пагинации.
func (r *AssignmentRepository) GetList(filter models.AssignmentFilter) (*models.PagedResult[models.Assignment], error) {
	if !filter.ShowFinished && len(filter.Statuses) > 0 {
		hasVisibleStatus := false
		for _, status := range filter.Statuses {
			if status != "finished" {
				hasVisibleStatus = true
				break
			}
		}
		if !hasVisibleStatus {
			return &models.PagedResult[models.Assignment]{Items: []models.Assignment{}, Page: filter.Page, PageSize: filter.PageSize}, nil
		}
	}
	query := `
		SELECT
			a.id, a.document_id, d.kind,
			COALESCE(a.executor_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(u_executor.full_name, ''),
			CASE WHEN a.type = 'execution' THEN a.content ELSE '' END, a.deadline, a.status, a.report, a.completed_at,
			a.series_id, a.iteration_number, a.planned_deadline,
			COALESCE(s.current_assignment_id = a.id, FALSE) AS is_series_current,
			a.created_at, a.updated_at,
			d.registration_number as doc_number,
			d.content as doc_subject, a.type, a.creator_id, COALESCE(u_creator.full_name, '')
		FROM assignments a
		JOIN documents d ON d.id = a.document_id
		LEFT JOIN users u_executor ON a.executor_id = u_executor.id
		LEFT JOIN users u_creator ON a.creator_id = u_creator.id
		LEFT JOIN assignment_series s ON s.id = a.series_id
	`

	where := []string{"(a.series_id IS NULL OR s.current_assignment_id = a.id)"}
	args := []interface{}{}
	argIdx := 1
	if len(filter.Types) > 0 {
		where = append(where, fmt.Sprintf("a.type = ANY($%d)", argIdx))
		args = append(args, pq.Array(filter.Types))
		argIdx++
	}

	if filter.DocumentID != "" {
		where = append(where, fmt.Sprintf("a.document_id = $%d", argIdx))
		args = append(args, filter.DocumentID)
		argIdx++
	}
	accessibleIDs := accessibleUserIDs(filter.AccessibleByUserID, filter.AccessibleByUserIDs)
	if len(filter.Types) != 1 || filter.Types[0] != models.AssignmentTypeExecution {
		applyTypedAssignmentAccess(&where, &args, &argIdx, filter)
	} else if filter.Mode == models.WorkspaceModeControl {
		controlClauses := make([]string, 0, len(filter.ControlScopes))
		for _, spec := range models.AllDocumentKindSpecs() {
			scope, ok := filter.ControlScopes[spec.Code]
			if !ok {
				continue
			}
			kindWhere := []string{fmt.Sprintf("d.kind = $%d", argIdx)}
			args = append(args, string(spec.Code))
			argIdx++
			applyDocumentListAccess(&kindWhere, &args, &argIdx, scope)
			controlClauses = append(controlClauses, "("+strings.Join(kindWhere, " AND ")+")")
		}
		if len(controlClauses) == 0 {
			where = append(where, "1=0")
		} else {
			where = append(where, "("+strings.Join(controlClauses, " OR ")+")")
		}
	} else if len(filter.AllowedDocumentKinds) > 0 || len(accessibleIDs) > 0 {
		accessClauses := make([]string, 0, 2)
		if len(filter.AllowedDocumentKinds) > 0 {
			accessClauses = append(accessClauses, fmt.Sprintf("d.kind = ANY($%d)", argIdx))
			args = append(args, pq.Array(filter.AllowedDocumentKinds))
			argIdx++
		}
		if len(accessibleIDs) > 0 {
			if len(filter.AccessibleByUserIDs) > 0 {
				accessClauses = append(accessClauses, fmt.Sprintf("(a.executor_id = ANY($%d::uuid[]) OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = ANY($%d::uuid[])))", argIdx, argIdx))
				args = append(args, pq.Array(accessibleIDs))
			} else {
				accessClauses = append(accessClauses, fmt.Sprintf("(a.executor_id = $%d OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $%d))", argIdx, argIdx))
				args = append(args, filter.AccessibleByUserID)
			}
			argIdx++
		}
		where = append(where, "("+strings.Join(accessClauses, " OR ")+")")
	}
	if filter.ExecutorID != "" {
		// Фильтр по основному исполнителю ИЛИ соисполнителю
		where = append(where, fmt.Sprintf("(a.executor_id = $%d OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = $%d) OR EXISTS (SELECT 1 FROM assignment_recipients ar WHERE ar.assignment_id=a.id AND ar.user_id=$%d))", argIdx, argIdx, argIdx))
		args = append(args, filter.ExecutorID)
		argIdx++
	}

	if filter.Mode == models.WorkspaceModeExecution {
		if filter.ShowFinished {
			where = append(where, "a.status IN ('new', 'in_progress', 'returned', 'completed', 'finished')")
		} else {
			where = append(where, "a.status IN ('new', 'in_progress', 'returned')")
		}
	} else if filter.Mode == models.WorkspaceModeControl {
		if filter.ShowFinished {
			where = append(where, "a.status IN ('new', 'in_progress', 'returned', 'completed', 'finished')")
		} else {
			where = append(where, "a.status IN ('new', 'in_progress', 'returned', 'completed')")
		}
	}

	if filter.OverdueOnly {
		where = append(where, "a.status IN ('new', 'in_progress', 'returned') AND a.deadline::date < CURRENT_DATE")
	}

	if len(filter.Statuses) > 0 {
		where = append(where, fmt.Sprintf("a.status = ANY($%d)", argIdx))
		args = append(args, pq.Array(filter.Statuses))
		argIdx++
	}
	if !filter.ShowFinished {
		where = append(where, fmt.Sprintf("a.status != $%d", argIdx))
		args = append(args, "finished")
		argIdx++
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("a.deadline::date >= $%d::date", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("a.deadline::date <= $%d::date", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}

	if filter.Search != "" {
		search := "%" + strings.ToLower(filter.Search) + "%"
		where = append(where, fmt.Sprintf("((a.type = 'execution' AND LOWER(a.content) LIKE $%d) OR LOWER(d.registration_number) LIKE $%d OR LOWER(d.content) LIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, search)
		argIdx++
	}

	query += " WHERE " + strings.Join(where, " AND ")

	// Запрос количества
	countQuery := "SELECT COUNT(*) FROM assignments a JOIN documents d ON d.id = a.document_id LEFT JOIN assignment_series s ON s.id = a.series_id WHERE " + strings.Join(where, " AND ")
	var totalCount int
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count assignments: %w", err)
	}

	// Пагинация по умолчанию
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// Пагинация
	query += fmt.Sprintf(" ORDER BY a.created_at DESC, a.id DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}
	defer rows.Close()

	items := make([]models.Assignment, 0)
	assignmentIDs := make([]uuid.UUID, 0)
	assignmentIndex := map[uuid.UUID]int{} // ID поручения -> индекс в items

	for rows.Next() {
		var a models.Assignment
		var deadline sql.NullTime
		var completedAt sql.NullTime
		var report sql.NullString
		var docNumber sql.NullString
		var docSubject sql.NullString
		var seriesID uuid.NullUUID
		var iterationNumber sql.NullInt64
		var plannedDeadline sql.NullTime
		var creatorID uuid.NullUUID

		if err := rows.Scan(
			&a.ID, &a.DocumentID, &a.DocumentKind,
			&a.ExecutorID, &a.ExecutorName,
			&a.Content, &deadline, &a.Status, &report, &completedAt,
			&seriesID, &iterationNumber, &plannedDeadline, &a.IsSeriesCurrent,
			&a.CreatedAt, &a.UpdatedAt,
			&docNumber, &docSubject, &a.Type, &creatorID, &a.CreatorName,
		); err != nil {
			return nil, err
		}

		if creatorID.Valid {
			a.CreatorID = creatorID.UUID
		}
		if deadline.Valid {
			a.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
		}
		if seriesID.Valid {
			a.SeriesID = &seriesID.UUID
		}
		if iterationNumber.Valid {
			a.IterationNumber = int(iterationNumber.Int64)
		}
		if plannedDeadline.Valid {
			a.PlannedDeadline = &plannedDeadline.Time
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

		assignmentIndex[a.ID] = len(items)
		assignmentIDs = append(assignmentIDs, a.ID)
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(assignmentIDs) > 0 {
		coExecQuery := `
			SELECT ce.assignment_id, u.id, u.login, u.last_name, u.first_name, u.patronymic, u.no_patronymic
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
			if err := ceRows.Scan(&assignmentID, &u.ID, &u.Login, &u.LastName, &u.FirstName, &u.Patronymic, &u.NoPatronymic); err != nil {
				return nil, fmt.Errorf("failed to scan co-executor: %w", err)
			}

			if idx, ok := assignmentIndex[assignmentID]; ok {
				items[idx].CoExecutors = append(items[idx].CoExecutors, u)
				items[idx].CoExecutorIDs = append(items[idx].CoExecutorIDs, u.ID.String())
			}
		}
		if err := ceRows.Err(); err != nil {
			return nil, err
		}
	}

	var recipientTaskIDs []uuid.UUID
	for _, item := range items {
		if item.Type == models.AssignmentTypeAcknowledgment {
			recipientTaskIDs = append(recipientTaskIDs, item.ID)
		}
	}
	if len(recipientTaskIDs) > 0 {
		recipients, err := r.GetRecipientsByAssignmentIDs(recipientTaskIDs)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].Users = recipients[items[i].ID]
		}
	}
	return &models.PagedResult[models.Assignment]{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

// HasDocumentAccess проверяет, есть ли у пользователя доступ к документу как у исполнителя или соисполнителя поручения.
func (r *AssignmentRepository) HasDocumentAccess(userID, documentID uuid.UUID) (bool, error) {
	var hasAccess bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM assignments a
			WHERE a.document_id = $1
			  AND (
				a.executor_id = $2
                OR EXISTS (SELECT 1 FROM assignment_recipients ar WHERE ar.assignment_id = a.id AND ar.user_id = $2)
				OR EXISTS (
					SELECT 1
					FROM assignment_co_executors ce
					WHERE ce.assignment_id = a.id AND ce.user_id = $2
				)
			  )
		)
	`

	if err := r.db.QueryRow(query, documentID, userID).Scan(&hasAccess); err != nil {
		return false, fmt.Errorf("failed to check document access by assignment: %w", err)
	}

	return hasAccess, nil
}

// GetAccessibleDocumentIDs возвращает документы из набора, доступные пользователю через поручения.
func (r *AssignmentRepository) GetAccessibleDocumentIDs(userID uuid.UUID, documentIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	result := make(map[uuid.UUID]struct{})
	if len(documentIDs) == 0 {
		return result, nil
	}

	idStrings := make([]string, 0, len(documentIDs))
	for _, documentID := range documentIDs {
		idStrings = append(idStrings, documentID.String())
	}

	rows, err := r.db.Query(`
		SELECT DISTINCT a.document_id
		FROM assignments a
		WHERE a.document_id = ANY($1::uuid[])
		  AND (
			a.executor_id = $2
                OR EXISTS (SELECT 1 FROM assignment_recipients ar WHERE ar.assignment_id = a.id AND ar.user_id = $2)
			OR EXISTS (
				SELECT 1
				FROM assignment_co_executors ce
				WHERE ce.assignment_id = a.id AND ce.user_id = $2
			)
		  )
	`, pq.Array(idStrings), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessible documents by assignment: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var documentID uuid.UUID
		if err := rows.Scan(&documentID); err != nil {
			return nil, err
		}
		result[documentID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Management of both types uses assign; participation never grants control.
func applyTypedAssignmentAccess(where *[]string, args *[]interface{}, argIdx *int, filter models.AssignmentFilter) {
	clauses := []string{}
	if filter.Mode != models.WorkspaceModeExecution {
		for _, typ := range []string{models.AssignmentTypeExecution, models.AssignmentTypeAcknowledgment} {
			scopes := filter.ControlScopes
			for _, spec := range models.AllDocumentKindSpecs() {
				scope, ok := scopes[spec.Code]
				if !ok {
					continue
				}
				part := []string{fmt.Sprintf("a.type = '%s' AND d.kind = $%d", typ, *argIdx)}
				*args = append(*args, string(spec.Code))
				*argIdx++
				applyDocumentListAccess(&part, args, argIdx, scope)
				clauses = append(clauses, "("+strings.Join(part, " AND ")+")")
			}
		}
	}
	if filter.Mode != models.WorkspaceModeControl {
		ids := accessibleUserIDs(filter.AccessibleByUserID, filter.AccessibleByUserIDs)
		if len(ids) > 0 {
			pending := ""
			if filter.Mode == models.WorkspaceModeExecution && !filter.ShowFinished {
				pending = " AND ar.confirmed_at IS NULL"
			}
			clauses = append(clauses, fmt.Sprintf(`(a.executor_id = ANY($%d::uuid[])
    OR EXISTS (SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id=a.id AND ce.user_id=ANY($%d::uuid[]))
    OR EXISTS (SELECT 1 FROM assignment_recipients ar WHERE ar.assignment_id=a.id AND ar.user_id=ANY($%d::uuid[])%s))`, *argIdx, *argIdx, *argIdx, pending))
			*args = append(*args, pq.Array(ids))
			*argIdx++
		}
	}
	if len(clauses) == 0 {
		*where = append(*where, "1=0")
	} else {
		*where = append(*where, "("+strings.Join(clauses, " OR ")+")")
	}
}

package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CreateSeriesWithFirstAssignment persists a recurring template, its first
// ordinary assignment and all observable effects as one unit.
func (r *AssignmentRepository) CreateSeriesWithFirstAssignment(
	seriesID, assignmentID, documentID, executorID, createdBy uuid.UUID,
	content string, firstDeadline time.Time, intervalUnit string, intervalValue int, dayRule string,
	dayOfMonth int, coExecutorIDs []string, effects []models.OutboxEvent,
) (*models.AssignmentSeries, error) {
	if r.outbox == nil {
		return nil, ErrOutboxNotConfigured
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var day any
	if dayRule == "fixed" {
		day = dayOfMonth
	}
	if _, err = tx.Exec(`
		INSERT INTO assignment_series
			(id, document_id, executor_id, content, interval_unit, interval_value, day_rule, day_of_month, current_iteration, active, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,1,TRUE,$9)
	`, seriesID, documentID, executorID, content, intervalUnit, intervalValue, dayRule, day, createdBy); err != nil {
		return nil, fmt.Errorf("failed to create assignment series: %w", err)
	}
	if _, err = tx.Exec(`
		INSERT INTO assignments
			(id, document_id, executor_id, content, deadline, planned_deadline, status, series_id, iteration_number)
		VALUES ($1,$2,$3,$4,$5,$5,'new',$6,1)
	`, assignmentID, documentID, executorID, content, firstDeadline, seriesID); err != nil {
		return nil, fmt.Errorf("failed to create first series assignment: %w", err)
	}
	if err = insertSeriesCoExecutors(tx, seriesID, coExecutorIDs); err != nil {
		return nil, err
	}
	if err = insertAssignmentCoExecutors(tx, assignmentID, coExecutorIDs); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`UPDATE assignment_series SET current_assignment_id=$1 WHERE id=$2`, assignmentID, seriesID); err != nil {
		return nil, err
	}
	for _, effect := range effects {
		if err = r.outbox.EnqueueTx(tx, effect); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAssignmentSeries(seriesID)
}

func insertSeriesCoExecutors(tx *sql.Tx, seriesID uuid.UUID, values []string) error {
	for _, value := range values {
		uid, err := uuid.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid co-executor ID %s: %w", value, err)
		}
		if _, err = tx.Exec(`INSERT INTO assignment_series_co_executors(series_id,user_id) VALUES($1,$2)`, seriesID, uid); err != nil {
			return err
		}
	}
	return nil
}

func insertAssignmentCoExecutors(tx *sql.Tx, assignmentID uuid.UUID, values []string) error {
	for _, value := range values {
		uid, err := uuid.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid co-executor ID %s: %w", value, err)
		}
		if _, err = tx.Exec(`INSERT INTO assignment_co_executors(assignment_id,user_id) VALUES($1,$2)`, assignmentID, uid); err != nil {
			return err
		}
	}
	return nil
}

func (r *AssignmentRepository) GetAssignmentSeries(id uuid.UUID) (*models.AssignmentSeries, error) {
	var value models.AssignmentSeries
	var currentID uuid.NullUUID
	var cancelledBy uuid.NullUUID
	var cancelledAt sql.NullTime
	var day sql.NullInt64
	err := r.db.QueryRow(`
		SELECT s.id,s.document_id,d.kind,COALESCE(d.registration_number,''),s.executor_id,
		       COALESCE(u.full_name,''),s.content,s.interval_unit,s.interval_value,s.day_rule,s.day_of_month,
		       s.current_assignment_id,s.current_iteration,s.active,s.created_by,s.cancelled_by,
		       s.cancelled_at,s.created_at,s.updated_at
		FROM assignment_series s
		JOIN documents d ON d.id=s.document_id
		LEFT JOIN users u ON u.id=s.executor_id
		WHERE s.id=$1
	`, id).Scan(&value.ID, &value.DocumentID, &value.DocumentKind, &value.DocumentNumber,
		&value.ExecutorID, &value.ExecutorName, &value.Content, &value.IntervalUnit, &value.IntervalValue,
		&value.DayRule, &day, &currentID, &value.CurrentIteration, &value.Active,
		&value.CreatedBy, &cancelledBy, &cancelledAt, &value.CreatedAt, &value.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if day.Valid {
		value.DayOfMonth = int(day.Int64)
	}
	if currentID.Valid {
		value.CurrentAssignmentID = &currentID.UUID
	}
	if cancelledBy.Valid {
		value.CancelledBy = &cancelledBy.UUID
	}
	if cancelledAt.Valid {
		value.CancelledAt = &cancelledAt.Time
	}
	rows, err := r.db.Query(`
		SELECT u.id,u.login,u.full_name
		FROM assignment_series_co_executors ce JOIN users u ON u.id=ce.user_id
		WHERE ce.series_id=$1 ORDER BY u.full_name
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var user models.User
		if err = rows.Scan(&user.ID, &user.Login, &user.FullName); err != nil {
			return nil, err
		}
		value.CoExecutors = append(value.CoExecutors, user)
		value.CoExecutorIDs = append(value.CoExecutorIDs, user.ID.String())
	}
	return &value, rows.Err()
}

func (r *AssignmentRepository) GetAssignmentSeriesByAssignment(id uuid.UUID) (*models.AssignmentSeries, error) {
	var seriesID uuid.NullUUID
	if err := r.db.QueryRow(`SELECT series_id FROM assignments WHERE id=$1`, id).Scan(&seriesID); err != nil {
		return nil, err
	}
	if !seriesID.Valid {
		return nil, nil
	}
	return r.GetAssignmentSeries(seriesID.UUID)
}

func (r *AssignmentRepository) UpdateAssignmentSeries(id, executorID uuid.UUID, content, intervalUnit string, intervalValue int, dayRule string, dayOfMonth int, coExecutorIDs []string, effects []models.OutboxEvent) (*models.AssignmentSeries, error) {
	if r.outbox == nil {
		return nil, ErrOutboxNotConfigured
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var day any
	if dayRule == "fixed" {
		day = dayOfMonth
	}
	result, err := tx.Exec(`UPDATE assignment_series SET executor_id=$1,content=$2,interval_unit=$3,interval_value=$4,day_rule=$5,day_of_month=$6,updated_at=NOW() WHERE id=$7 AND active=TRUE`, executorID, content, intervalUnit, intervalValue, dayRule, day, id)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, models.NewConflict("отменённую серию нельзя редактировать")
	}
	if _, err = tx.Exec(`DELETE FROM assignment_series_co_executors WHERE series_id=$1`, id); err != nil {
		return nil, err
	}
	if err = insertSeriesCoExecutors(tx, id, coExecutorIDs); err != nil {
		return nil, err
	}
	for _, effect := range effects {
		if err = r.outbox.EnqueueTx(tx, effect); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAssignmentSeries(id)
}

func (r *AssignmentRepository) CancelAssignmentSeries(id, actorID uuid.UUID, effects []models.OutboxEvent) error {
	if r.outbox == nil {
		return ErrOutboxNotConfigured
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE assignment_series SET active=FALSE,cancelled_by=$1,cancelled_at=NOW(),updated_at=NOW() WHERE id=$2 AND active=TRUE`, actorID, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return models.NewConflict("серия уже отменена")
	}
	for _, effect := range effects {
		if err = r.outbox.EnqueueTx(tx, effect); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// FinishSeriesIterationWithNext accepts the current assignment and advances
// the series in one transaction. A concurrent retry cannot create a duplicate
// because both the current pointer and the unique iteration key are checked.
func (r *AssignmentRepository) FinishSeriesIterationWithNext(
	currentID, seriesID, nextID uuid.UUID, expectedSeriesUpdatedAt time.Time,
	report string, completedAt *time.Time, nextDeadline time.Time, nextIteration int, executorID uuid.UUID, content string,
	coExecutorIDs []string, currentEffects, nextEffects []models.OutboxEvent,
) (*models.Assignment, error) {
	if r.outbox == nil {
		return nil, ErrOutboxNotConfigured
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var active bool
	var pointer uuid.NullUUID
	var actualUpdatedAt time.Time
	if err = tx.QueryRow(`SELECT active,current_assignment_id,updated_at FROM assignment_series WHERE id=$1 FOR UPDATE`, seriesID).Scan(&active, &pointer, &actualUpdatedAt); err != nil {
		return nil, err
	}
	if !actualUpdatedAt.Equal(expectedSeriesUpdatedAt) {
		return nil, models.NewConflict("параметры серии были изменены; повторите принятие поручения")
	}
	if !pointer.Valid || pointer.UUID != currentID {
		return nil, models.NewConflict("итерация больше не является текущей")
	}
	result, err := tx.Exec(`UPDATE assignments SET status='finished',report=$1,completed_at=$2,updated_at=NOW() WHERE id=$3 AND status='completed'`, report, completedAt, currentID)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, models.NewConflict("поручение уже изменено")
	}
	for _, effect := range currentEffects {
		if err = r.outbox.EnqueueTx(tx, effect); err != nil {
			return nil, err
		}
	}
	if active {
		if _, err = tx.Exec(`INSERT INTO assignments(id,document_id,executor_id,content,deadline,planned_deadline,status,series_id,iteration_number) SELECT $1,document_id,$2,$3,$4,$4,'new',id,$5 FROM assignment_series WHERE id=$6`, nextID, executorID, content, nextDeadline, nextIteration, seriesID); err != nil {
			return nil, err
		}
		if err = insertAssignmentCoExecutors(tx, nextID, coExecutorIDs); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(`UPDATE assignment_series SET current_assignment_id=$1,current_iteration=$2,updated_at=NOW() WHERE id=$3`, nextID, nextIteration, seriesID); err != nil {
			return nil, err
		}
		for _, effect := range nextEffects {
			if err = r.outbox.EnqueueTx(tx, effect); err != nil {
				return nil, err
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(currentID)
}

func (r *AssignmentRepository) GetAssignmentSeriesHistory(seriesID uuid.UUID) ([]models.Assignment, error) {
	rows, err := r.db.Query(`
		SELECT a.id,a.document_id,d.kind,a.executor_id,COALESCE(u.full_name,''),a.content,
		       a.deadline,a.status,a.report,a.completed_at,a.series_id,a.iteration_number,
		       a.planned_deadline,COALESCE(s.current_assignment_id=a.id,FALSE),a.created_at,
		       a.updated_at,COALESCE(d.registration_number,''),COALESCE(d.content,'')
		FROM assignments a
		JOIN documents d ON d.id=a.document_id
		JOIN assignment_series s ON s.id=a.series_id
		LEFT JOIN users u ON u.id=a.executor_id
		WHERE a.series_id=$1 ORDER BY a.iteration_number DESC
	`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.Assignment, 0)
	ids := make([]uuid.UUID, 0)
	index := map[uuid.UUID]int{}
	for rows.Next() {
		var a models.Assignment
		var deadline, completedAt, planned sql.NullTime
		var report sql.NullString
		var sid uuid.NullUUID
		var iteration sql.NullInt64
		if err = rows.Scan(&a.ID, &a.DocumentID, &a.DocumentKind, &a.ExecutorID, &a.ExecutorName,
			&a.Content, &deadline, &a.Status, &report, &completedAt, &sid, &iteration,
			&planned, &a.IsSeriesCurrent, &a.CreatedAt, &a.UpdatedAt, &a.DocumentNumber,
			&a.DocumentSubject); err != nil {
			return nil, err
		}
		if deadline.Valid {
			a.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
		}
		if planned.Valid {
			a.PlannedDeadline = &planned.Time
		}
		if report.Valid {
			a.Report = report.String
		}
		if sid.Valid {
			a.SeriesID = &sid.UUID
		}
		if iteration.Valid {
			a.IterationNumber = int(iteration.Int64)
		}
		index[a.ID] = len(items)
		ids = append(ids, a.ID)
		items = append(items, a)
	}
	if err = rows.Err(); err != nil || len(ids) == 0 {
		return items, err
	}
	coRows, err := r.db.Query(`SELECT ce.assignment_id,u.id,u.login,u.full_name FROM assignment_co_executors ce JOIN users u ON u.id=ce.user_id WHERE ce.assignment_id=ANY($1)`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer coRows.Close()
	for coRows.Next() {
		var assignmentID uuid.UUID
		var user models.User
		if err = coRows.Scan(&assignmentID, &user.ID, &user.Login, &user.FullName); err != nil {
			return nil, err
		}
		if i, ok := index[assignmentID]; ok {
			items[i].CoExecutors = append(items[i].CoExecutors, user)
			items[i].CoExecutorIDs = append(items[i].CoExecutorIDs, user.ID.String())
		}
	}
	return items, coRows.Err()
}

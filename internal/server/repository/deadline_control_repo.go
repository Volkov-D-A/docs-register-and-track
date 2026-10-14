package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/effects"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type DeadlineControlRepository struct {
	db     *database.DB
	outbox *OutboxRepository
}

func NewDeadlineControlRepository(db *database.DB, outbox *OutboxRepository) *DeadlineControlRepository {
	return &DeadlineControlRepository{db: db, outbox: outbox}
}

// Every seed must refer to the authenticated, active user. UNION terminates cycles
// and deduplicates multiple assignment and link paths before any aggregate/limit.
func visibleDocumentClosure(user string) string {
	return `WITH RECURSIVE visible(id) AS (
 SELECT d.id FROM documents d JOIN users u ON u.id=` + user + ` AND u.is_active
 WHERE EXISTS(SELECT 1 FROM document_permissions p WHERE p.kind_code=d.kind AND p.is_allowed
   AND p.action IN ('register','control_deadlines')
   AND ((p.subject_type='user' AND p.subject_key=u.id::text)
     OR (p.subject_type='department' AND p.subject_key=u.department_id::text)))
 OR EXISTS(SELECT 1 FROM department_nomenclature n WHERE n.department_id=u.department_id AND n.nomenclature_id=d.nomenclature_id)
 OR EXISTS(SELECT 1 FROM document_deadline_controls c JOIN deadline_executors e
   ON (e.id=c.executor_id OR e.id=ANY(c.co_executor_ids))
   WHERE c.document_id=d.id AND c.status IN ('active','completed') AND e.user_id=u.id)
 UNION
 SELECT CASE WHEN l.source_document_id=v.id THEN l.target_document_id ELSE l.source_document_id END
 FROM visible v JOIN document_links l ON (l.source_document_id=v.id OR l.target_document_id=v.id)
 ) `
}

const deadlineManagePredicate = `EXISTS(SELECT 1 FROM users u JOIN document_permissions p
 ON ((p.subject_type='user' AND p.subject_key=u.id::text) OR (p.subject_type='department' AND p.subject_key=u.department_id::text))
 WHERE u.id=$1 AND u.is_active AND p.kind_code=d.kind AND p.action='control_deadlines' AND p.is_allowed)`

func (r *DeadlineControlRepository) CanRead(ctx context.Context, actor uuid.UUID, documentID string) (bool, error) {
	var result bool
	err := r.db.QueryRowContext(ctx, visibleDocumentClosure("$1")+`SELECT EXISTS(SELECT 1 FROM visible WHERE id=$2)`, actor, documentID).Scan(&result)
	return result, err
}
func (r *DeadlineControlRepository) CanManage(ctx context.Context, actor uuid.UUID, documentID string) (bool, error) {
	var result bool
	err := r.db.QueryRowContext(ctx, visibleDocumentClosure("$1")+`SELECT EXISTS(SELECT 1 FROM documents d JOIN visible v ON v.id=d.id WHERE d.id=$2 AND `+deadlineManagePredicate+`)`, actor, documentID).Scan(&result)
	return result, err
}

const deadlineControlColumns = `c.id::text,c.document_id::text,d.kind,COALESCE(d.registration_number,''),c.created_by::text,
 c.executor_id::text,c.co_executor_ids,c.subject,to_char(c.due_date,'YYYY-MM-DD'),c.status,
 COALESCE(to_char(c.executed_on,'YYYY-MM-DD'),''),c.completed_at,COALESCE(c.completed_by::text,''),c.comment,c.cancellation_reason,
 COALESCE(c.series_id::text,''),c.iteration,COALESCE(to_char(c.planned_due_date,'YYYY-MM-DD'),''),c.version,c.created_at,c.updated_at`

type deadlineScanner interface{ Scan(...any) error }

func scanDeadlineControl(row deadlineScanner) (*models.DocumentDeadlineControl, error) {
	var c models.DocumentDeadlineControl
	co := pq.StringArray{}
	err := row.Scan(&c.ID, &c.DocumentID, &c.DocumentKind, &c.DocumentNumber, &c.CreatedBy, &c.ExecutorID, &co, &c.Subject, &c.DueDate, &c.Status, &c.ExecutedOn, &c.CompletedAt, &c.CompletedBy, &c.Comment, &c.CancellationReason, &c.SeriesID, &c.Iteration, &c.PlannedDueDate, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	c.CoExecutorIDs = []string(co)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.NewNotFound("контроль не найден")
	}
	return &c, err
}
func (r *DeadlineControlRepository) GetControl(ctx context.Context, id string) (*models.DocumentDeadlineControl, error) {
	return scanDeadlineControl(r.db.QueryRowContext(ctx, `SELECT `+deadlineControlColumns+` FROM document_deadline_controls c JOIN documents d ON d.id=c.document_id WHERE c.id=$1`, id))
}

const deadlineSeriesColumns = `id::text,document_id::text,created_by::text,executor_id::text,co_executor_ids,subject,
 interval_unit,interval_value,day_rule,day_of_month,to_char(first_due_date,'YYYY-MM-DD'),active,stop_reason,
 COALESCE(current_control_id::text,''),current_iteration,version`

func scanDeadlineSeries(row deadlineScanner) (*models.DocumentDeadlineSeries, error) {
	var s models.DocumentDeadlineSeries
	co := pq.StringArray{}
	err := row.Scan(&s.ID, &s.DocumentID, &s.CreatedBy, &s.ExecutorID, &co, &s.Subject, &s.Recurrence.IntervalUnit, &s.Recurrence.IntervalValue, &s.Recurrence.DayRule, &s.Recurrence.DayOfMonth, &s.FirstDueDate, &s.Active, &s.StopReason, &s.CurrentControlID, &s.CurrentIteration, &s.Version)
	s.CoExecutorIDs = []string(co)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.NewNotFound("серия не найдена")
	}
	return &s, err
}
func (r *DeadlineControlRepository) GetSeries(ctx context.Context, id string) (*models.DocumentDeadlineSeries, error) {
	return scanDeadlineSeries(r.db.QueryRowContext(ctx, `SELECT `+deadlineSeriesColumns+` FROM document_deadline_series WHERE id=$1`, id))
}

func (r *DeadlineControlRepository) ListControls(ctx context.Context, actor uuid.UUID, f models.DeadlineControlFilter) (*models.DeadlineControlPage, error) {
	page, size := normalizePagination(f.Page, f.PageSize)
	if page < 1 {
		page = 1
	}
	result := &models.DeadlineControlPage{Items: []models.DocumentDeadlineControl{}, Page: page, PageSize: size}
	args := []any{actor}
	where := []string{}
	for _, item := range []struct{ column, value string }{{"c.document_id", f.DocumentID}, {"c.series_id", f.SeriesID}, {"c.status", f.Status}, {"c.due_date >=", f.DueFrom}, {"c.due_date <=", f.DueTo}} {
		if item.value == "" {
			continue
		}
		operator := " = "
		column := item.column
		if strings.HasSuffix(column, " >=") || strings.HasSuffix(column, " <=") {
			column = item.column[:len(item.column)-3]
			operator = " " + item.column[len(item.column)-2:] + " "
		}
		args = append(args, item.value)
		where = append(where, fmt.Sprintf("%s%s$%d", column, operator, len(args)))
	}
	if f.ManageOnly {
		where = append(where, deadlineManagePredicate)
	}
	if f.SeriesStoppedOnly {
		where = append(where, `EXISTS(SELECT 1 FROM document_deadline_series s WHERE s.id=c.series_id AND s.current_control_id=c.id AND NOT s.active)`)
	}
	if f.MineOnly {
		where = append(where, `EXISTS(SELECT 1 FROM deadline_executors e WHERE e.user_id=$1 AND (e.id=c.executor_id OR e.id=ANY(c.co_executor_ids)))`)
	}
	if f.OverdueOnly {
		args = append(args, models.OrganizationToday(time.Now()))
		where = append(where, fmt.Sprintf("c.status='active' AND c.due_date<$%d", len(args)))
	}
	base := ` FROM document_deadline_controls c JOIN documents d ON d.id=c.document_id JOIN visible v ON v.id=c.document_id`
	if len(where) > 0 {
		base += ` WHERE ` + strings.Join(where, " AND ")
	}
	tx, err := r.db.SQLDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	prefix := visibleDocumentClosure("$1")
	if err = tx.QueryRowContext(ctx, prefix+`SELECT COUNT(*)`+base, args...).Scan(&result.Total); err != nil {
		return nil, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := tx.QueryContext(ctx, prefix+`SELECT `+deadlineControlColumns+`,`+deadlineManagePredicate+base+fmt.Sprintf(` ORDER BY c.due_date,c.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		// Keep one scanner for both list and single-record reads.
		scanner := &deadlineControlWithPermission{rows: rows}
		item, err := scanDeadlineControl(scanner)
		if err != nil {
			return nil, err
		}
		item.CanManage = scanner.canManage
		result.Items = append(result.Items, *item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	return result, tx.Commit()
}

type deadlineControlWithPermission struct {
	rows      *sql.Rows
	canManage bool
}

func (s *deadlineControlWithPermission) Scan(args ...any) error {
	return s.rows.Scan(append(args, &s.canManage)...)
}

// A document row is the shared lock for all controls and series on that document.
// This ordering makes completion, stopping, resumption and editing serializable
// without relying on an in-process mutex or on reading a template before locking.
func (r *DeadlineControlRepository) WithDocument(ctx context.Context, actor uuid.UUID, doc string, fn func(ports.DeadlineTransaction) error) error {
	if r.outbox == nil {
		return ErrOutboxNotConfigured
	}
	tx, err := r.db.SQLDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var active bool
	if err = tx.QueryRowContext(ctx, `SELECT is_active FROM users WHERE id=$1 FOR SHARE`, actor).Scan(&active); err != nil {
		return err
	}
	if !active {
		return models.ErrForbidden
	}
	var kind, number string
	err = tx.QueryRowContext(ctx, `SELECT kind,COALESCE(registration_number,'') FROM documents WHERE id=$1 FOR UPDATE`, doc).Scan(&kind, &number)
	if errors.Is(err, sql.ErrNoRows) {
		return models.NewNotFound("документ не найден")
	}
	if err != nil {
		return err
	}
	var permissionID uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT p.id FROM document_permissions p JOIN users u ON u.id=$1
 WHERE p.kind_code=$2 AND p.action='control_deadlines' AND p.is_allowed
 AND ((p.subject_type='user' AND p.subject_key=u.id::text) OR (p.subject_type='department' AND p.subject_key=u.department_id::text))
 ORDER BY p.id LIMIT 1 FOR SHARE OF p`, actor, kind).Scan(&permissionID)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ErrForbidden
	}
	if err != nil {
		return err
	}
	var allowed bool
	err = tx.QueryRowContext(ctx, visibleDocumentClosure("$1")+`SELECT EXISTS(SELECT 1 FROM documents d JOIN visible v ON v.id=d.id WHERE d.id=$2 AND `+deadlineManagePredicate+`)`, actor, doc).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}
	work := &deadlineTransaction{ctx: ctx, tx: tx, outbox: r.outbox, doc: doc, kind: kind, number: number}
	if err = fn(work); err != nil {
		return err
	}
	return tx.Commit()
}

type deadlineTransaction struct {
	ctx               context.Context
	tx                *sql.Tx
	outbox            *OutboxRepository
	doc, kind, number string
}

func (t *deadlineTransaction) GetControl(id string) (*models.DocumentDeadlineControl, error) {
	return scanDeadlineControl(t.tx.QueryRowContext(t.ctx, `SELECT `+deadlineControlColumns+` FROM document_deadline_controls c JOIN documents d ON d.id=c.document_id WHERE c.id=$1 AND c.document_id=$2 FOR UPDATE OF c`, id, t.doc))
}
func (t *deadlineTransaction) GetSeries(id string) (*models.DocumentDeadlineSeries, error) {
	return scanDeadlineSeries(t.tx.QueryRowContext(t.ctx, `SELECT `+deadlineSeriesColumns+` FROM document_deadline_series WHERE id=$1 AND document_id=$2 FOR UPDATE`, id, t.doc))
}
func nullableDeadlineValue(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func (t *deadlineTransaction) SaveControl(c *models.DocumentDeadlineControl, create bool) error {
	if c.DocumentID != t.doc {
		return models.ErrForbidden
	}
	if create {
		err := t.tx.QueryRowContext(t.ctx, `INSERT INTO document_deadline_controls(id,document_id,created_by,executor_id,co_executor_ids,subject,due_date,series_id,iteration,planned_due_date)
   VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING version,created_at,updated_at`, c.ID, c.DocumentID, c.CreatedBy, c.ExecutorID, pq.Array(c.CoExecutorIDs), c.Subject, c.DueDate, nullableDeadlineValue(c.SeriesID), c.Iteration, nullableDeadlineValue(c.PlannedDueDate)).Scan(&c.Version, &c.CreatedAt, &c.UpdatedAt)
		c.DocumentKind = t.kind
		c.DocumentNumber = t.number
		return err
	}
	err := t.tx.QueryRowContext(t.ctx, `UPDATE document_deadline_controls SET executor_id=$1,co_executor_ids=$2,subject=$3,due_date=$4,status=$5,
 executed_on=$6,completed_at=$7,completed_by=$8,comment=$9,cancellation_reason=$10,version=version+1,updated_at=now()
 WHERE id=$11 AND document_id=$12 AND version=$13 AND status='active' RETURNING version,updated_at`, c.ExecutorID, pq.Array(c.CoExecutorIDs), c.Subject, c.DueDate, c.Status, nullableDeadlineValue(c.ExecutedOn), c.CompletedAt, nullableDeadlineValue(c.CompletedBy), c.Comment, c.CancellationReason, c.ID, t.doc, c.Version).Scan(&c.Version, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.NewConflict("контроль изменён или уже закрыт; перечитайте данные")
	}
	return err
}
func (t *deadlineTransaction) SaveSeries(s *models.DocumentDeadlineSeries, create bool) error {
	if s.DocumentID != t.doc {
		return models.ErrForbidden
	}
	if create {
		return t.tx.QueryRowContext(t.ctx, `INSERT INTO document_deadline_series(id,document_id,created_by,executor_id,co_executor_ids,subject,interval_unit,interval_value,day_rule,day_of_month,first_due_date,current_control_id,current_iteration)
   VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING version`, s.ID, s.DocumentID, s.CreatedBy, s.ExecutorID, pq.Array(s.CoExecutorIDs), s.Subject, s.Recurrence.IntervalUnit, s.Recurrence.IntervalValue, s.Recurrence.DayRule, s.Recurrence.DayOfMonth, s.FirstDueDate, nullableDeadlineValue(s.CurrentControlID), s.CurrentIteration).Scan(&s.Version)
	}
	err := t.tx.QueryRowContext(t.ctx, `UPDATE document_deadline_series SET executor_id=$1,co_executor_ids=$2,subject=$3,interval_unit=$4,interval_value=$5,day_rule=$6,day_of_month=$7,
 active=$8,stop_reason=$9,current_control_id=$10,current_iteration=$11,version=version+1 WHERE id=$12 AND document_id=$13 AND version=$14 RETURNING version`, s.ExecutorID, pq.Array(s.CoExecutorIDs), s.Subject, s.Recurrence.IntervalUnit, s.Recurrence.IntervalValue, s.Recurrence.DayRule, s.Recurrence.DayOfMonth, s.Active, s.StopReason, nullableDeadlineValue(s.CurrentControlID), s.CurrentIteration, s.ID, t.doc, s.Version).Scan(&s.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return models.NewConflict("серия изменена; перечитайте данные")
	}
	return err
}
func (t *deadlineTransaction) EligibleExecutors(ids []string) (bool, error) {
	userRows, err := t.tx.QueryContext(t.ctx, `SELECT u.id FROM users u JOIN deadline_executors e ON e.user_id=u.id WHERE e.id=ANY($1::uuid[]) ORDER BY u.id FOR SHARE OF u`, pq.Array(ids))
	if err != nil {
		return false, err
	}
	for userRows.Next() {
		var id uuid.UUID
		if err = userRows.Scan(&id); err != nil {
			userRows.Close()
			return false, err
		}
	}
	err = userRows.Err()
	userRows.Close()
	if err != nil {
		return false, err
	}
	rows, err := t.tx.QueryContext(t.ctx, `SELECT e.id FROM deadline_executors e LEFT JOIN users u ON u.id=e.user_id
 WHERE e.id=ANY($1::uuid[]) AND e.is_active AND (e.user_id IS NULL OR (u.is_active AND u.is_document_participant)) FOR SHARE OF e`, pq.Array(ids))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return false, err
		}
		seen[id] = true
	}
	if err = rows.Err(); err != nil {
		return false, err
	}
	for _, id := range ids {
		if !seen[id] {
			return false, nil
		}
	}
	return true, nil
}
func (t *deadlineTransaction) Emit(events []models.OutboxEvent) error {
	return enqueueOutboxEffects(t.outbox, t.tx, events)
}
func (t *deadlineTransaction) RecipientUserIDs(ids []string) ([]uuid.UUID, error) {
	rows, err := t.tx.QueryContext(t.ctx, `SELECT DISTINCT u.id FROM deadline_executors e JOIN users u ON u.id=e.user_id WHERE e.id=ANY($1::uuid[]) AND u.is_active`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		values = append(values, id)
	}
	return values, rows.Err()
}
func (t *deadlineTransaction) ControllerUserIDs() ([]uuid.UUID, error) {
	rows, err := t.tx.QueryContext(t.ctx, `SELECT DISTINCT u.id FROM users u JOIN document_permissions p ON ((p.subject_type='user' AND p.subject_key=u.id::text) OR (p.subject_type='department' AND p.subject_key=u.department_id::text)) WHERE u.is_active AND p.is_allowed AND p.kind_code=$1 AND p.action='control_deadlines'`, t.kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		values = append(values, id)
	}
	return values, rows.Err()
}

const executorColumns = `e.id::text,COALESCE(e.user_id::text,''),COALESCE(u.full_name,e.name),e.institution,e.position,(e.is_active AND (e.user_id IS NULL OR (u.is_active AND u.is_document_participant))),e.version`

func scanDeadlineExecutor(row deadlineScanner) (*models.DeadlineExecutor, error) {
	var e models.DeadlineExecutor
	err := row.Scan(&e.ID, &e.UserID, &e.Name, &e.Institution, &e.Position, &e.IsActive, &e.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.NewNotFound("исполнитель не найден")
	}
	return &e, err
}
func (r *DeadlineControlRepository) ListExecutors(ctx context.Context, includeInactive bool) ([]models.DeadlineExecutor, error) {
	query := `SELECT ` + executorColumns + ` FROM deadline_executors e LEFT JOIN users u ON u.id=e.user_id`
	if !includeInactive {
		query += ` WHERE e.is_active AND (e.user_id IS NULL OR (u.is_active AND u.is_document_participant))`
	}
	query += ` ORDER BY COALESCE(u.full_name,e.name),e.id`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []models.DeadlineExecutor{}
	for rows.Next() {
		e, err := scanDeadlineExecutor(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, *e)
	}
	return values, rows.Err()
}
func (r *DeadlineControlRepository) SaveExternalExecutor(ctx context.Context, actor uuid.UUID, id string, req models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if r.outbox == nil {
		return nil, ErrOutboxNotConfigured
	}
	tx, err := r.db.SQLDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var allowed bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN user_system_permissions p ON p.user_id=u.id WHERE u.id=$1 AND u.is_active AND p.is_allowed AND p.permission IN ('admin','references'))`, actor).Scan(&allowed)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, models.ErrForbidden
	}
	var old *models.DeadlineExecutor
	var result *models.DeadlineExecutor
	if id == "" {
		if req.Version != 0 {
			return nil, models.NewBadRequest("версия новой записи должна быть нулевой")
		}
		id = uuid.NewString()
		_, err = tx.ExecContext(ctx, `INSERT INTO deadline_executors(id,name,institution,position,is_active) VALUES($1,$2,$3,$4,$5)`, id, req.Name, req.Institution, req.Position, req.IsActive)
	} else {
		old, err = scanDeadlineExecutor(tx.QueryRowContext(ctx, `SELECT `+executorColumns+` FROM deadline_executors e LEFT JOIN users u ON u.id=e.user_id WHERE e.id=$1 FOR UPDATE OF e`, id))
		if err != nil {
			return nil, err
		}
		if old.UserID != "" {
			return nil, models.NewBadRequest("учётный исполнитель изменяется только в карточке пользователя")
		}
		if old.Version != req.Version {
			return nil, models.NewConflict("исполнитель изменён; перечитайте справочник")
		}
		_, err = tx.ExecContext(ctx, `UPDATE deadline_executors SET name=$1,institution=$2,position=$3,is_active=$4,version=version+1 WHERE id=$5`, req.Name, req.Institution, req.Position, req.IsActive, id)
	}
	if err != nil {
		return nil, err
	}
	result, err = scanDeadlineExecutor(tx.QueryRowContext(ctx, `SELECT `+executorColumns+` FROM deadline_executors e LEFT JOIN users u ON u.id=e.user_id WHERE e.id=$1`, id))
	if err != nil {
		return nil, err
	}
	details, err := json.Marshal(struct{ Before, After *models.DeadlineExecutor }{old, result})
	if err != nil {
		return nil, err
	}
	event, err := effects.NewAdminAuditOutboxEvent(fmt.Sprintf("deadline-executor:%s:%d", id, result.Version), models.CreateAdminAuditLogRequest{UserID: actor, Action: "DEADLINE_EXECUTOR_SAVE", Details: string(details)})
	if err != nil {
		return nil, err
	}
	if err = enqueueOutboxEffects(r.outbox, tx, []models.OutboxEvent{event}); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

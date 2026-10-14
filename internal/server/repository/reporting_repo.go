package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

type ReportingRepository struct {
	db     *database.DB
	userID uuid.UUID
}

func NewReportingRepository(db *database.DB) *ReportingRepository {
	return &ReportingRepository{db: db}
}

func (r *ReportingRepository) ForUser(id uuid.UUID) *ReportingRepository {
	return &ReportingRepository{db: r.db, userID: id}
}
func (r *ReportingRepository) readPredicate(alias string) string {
	return documentReadPredicate("'"+r.userID.String()+"'::uuid", alias)
}
func reportPeriodColumn(req models.ReportRequest) string {
	switch req.PeriodBasis {
	case "due_date":
		return "a.due_date"
	case "executed_on":
		return "a.executed_on"
	default:
		return "d.registration_date"
	}
}

func (r *ReportingRepository) Run(ctx context.Context, req models.ReportRequest, start, end time.Time) ([]models.ReportRow, error) {
	var query string
	var args []any
	zone := req.Timezone
	if zone == "" {
		zone = models.OrganizationTimezone
	}
	switch req.Template {
	case "organizations":
		query = `WITH letters AS (
			SELECT DISTINCT d.id, cr.correspondent_org_id AS organization_id, 'incoming' AS direction
			FROM documents d JOIN document_correspondent_registrations cr ON cr.document_id = d.id
			WHERE d.kind = 'incoming_letter' AND ` + r.documentPeriodPredicate(req) + ` AND ` + r.readPredicate("d") + `
				AND ($4::text IS NULL OR d.kind=$4) AND ($5::uuid IS NULL OR d.nomenclature_id=$5::uuid) AND ($6::uuid IS NULL OR d.created_by=$6::uuid)
			UNION ALL
			SELECT d.id, od.recipient_org_id, 'outgoing'
			FROM documents d JOIN outgoing_document_details od ON od.document_id = d.id
			WHERE d.kind = 'outgoing_letter' AND ` + r.documentPeriodPredicate(req) + ` AND ` + r.readPredicate("d") + `
				AND ($4::text IS NULL OR d.kind=$4) AND ($5::uuid IS NULL OR d.nomenclature_id=$5::uuid) AND ($6::uuid IS NULL OR d.created_by=$6::uuid)
		)
		SELECT o.id::text, o.name,
			COUNT(DISTINCT l.id) FILTER (WHERE l.direction = 'incoming')::int,
			COUNT(DISTINCT l.id) FILTER (WHERE l.direction = 'outgoing')::int
		FROM letters l JOIN organizations o ON o.id = l.organization_id
		WHERE ($3::uuid IS NULL OR o.id = $3::uuid)
		GROUP BY o.id, o.name ORDER BY o.name`
		args = []any{start, end, nullableID(req.OrganizationID), nullableStringValue(req.KindCode), nullableID(req.NomenclatureID), nullableID(req.UserID)}
	case "department_load":
		query = `SELECT COALESCE(dep.id::text, ''), COALESCE(dep.name, 'Без подразделения'),
			to_char(date_trunc('month', a.created_at AT TIME ZONE $4), 'YYYY-MM'), COUNT(*)::int
		FROM document_deadline_controls a JOIN deadline_executors e ON e.id=a.executor_id LEFT JOIN users u ON u.id = e.user_id
		JOIN documents d ON d.id=a.document_id
		LEFT JOIN departments dep ON dep.id = u.department_id
		WHERE ` + reportPeriodColumn(req) + ` >= $1::date AND ` + reportPeriodColumn(req) + ` <= $2::date
			AND $4::text = 'Asia/Yekaterinburg' AND ($3::uuid IS NULL OR dep.id = $3::uuid)
			AND ($5::text IS NULL OR d.kind=$5) AND ($6::uuid IS NULL OR d.nomenclature_id=$6::uuid)
			AND ($7::uuid IS NULL OR a.executor_id=$7::uuid) AND ($8::text IS NULL OR a.status=$8)
			AND ` + r.readPredicate("d") + `
		GROUP BY dep.id, dep.name, date_trunc('month', a.created_at AT TIME ZONE $4)
		ORDER BY date_trunc('month', a.created_at AT TIME ZONE $4), dep.name`
		args = deadlineReportArgs(req, start, end, zone)
	case "execution_time":
		query = `SELECT COALESCE(dep.id::text, ''), COALESCE(dep.name, 'Без подразделения'),
			COUNT(*)::int,
			ROUND(AVG(a.executed_on - (a.created_at AT TIME ZONE $4)::date)::numeric, 1)::double precision
		FROM document_deadline_controls a JOIN deadline_executors e ON e.id=a.executor_id LEFT JOIN users u ON u.id = e.user_id
		JOIN documents d ON d.id=a.document_id
		LEFT JOIN departments dep ON dep.id = u.department_id
		WHERE a.status='completed' AND ` + reportPeriodColumn(req) + ` >= $1::date AND ` + reportPeriodColumn(req) + ` <= $2::date
			AND $4::text = 'Asia/Yekaterinburg' AND ($3::uuid IS NULL OR dep.id = $3::uuid)
			AND ($5::text IS NULL OR d.kind=$5) AND ($6::uuid IS NULL OR d.nomenclature_id=$6::uuid)
			AND ($7::uuid IS NULL OR a.executor_id=$7::uuid) AND ($8::text IS NULL OR a.status=$8)
			AND ` + r.readPredicate("d") + `
		GROUP BY dep.id, dep.name ORDER BY dep.name`
		args = deadlineReportArgs(req, start, end, zone)
	case "overdue_rate":
		query = `SELECT COALESCE(dep.id::text, ''), COALESCE(dep.name, 'Без подразделения'),
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE a.executed_on > a.due_date)::int,
			ROUND((100.0 * COUNT(*) FILTER (WHERE a.executed_on > a.due_date) / COUNT(*))::numeric, 1)::double precision
		FROM document_deadline_controls a JOIN deadline_executors e ON e.id=a.executor_id LEFT JOIN users u ON u.id = e.user_id
		JOIN documents d ON d.id=a.document_id
		LEFT JOIN departments dep ON dep.id = u.department_id
		WHERE a.status='completed'
			AND ` + reportPeriodColumn(req) + ` >= $1::date AND ` + reportPeriodColumn(req) + ` <= $2::date
			AND $4::text = 'Asia/Yekaterinburg' AND ($3::uuid IS NULL OR dep.id = $3::uuid)
			AND ($5::text IS NULL OR d.kind=$5) AND ($6::uuid IS NULL OR d.nomenclature_id=$6::uuid)
			AND ($7::uuid IS NULL OR a.executor_id=$7::uuid) AND ($8::text IS NULL OR a.status=$8)
			AND ` + r.readPredicate("d") + `
		GROUP BY dep.id, dep.name ORDER BY dep.name`
		args = deadlineReportArgs(req, start, end, zone)
	default:
		return nil, fmt.Errorf("unknown report template %q", req.Template)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query report: %w", err)
	}
	defer rows.Close()
	result := make([]models.ReportRow, 0)
	for rows.Next() {
		var item models.ReportRow
		switch req.Template {
		case "organizations":
			err = rows.Scan(&item.Key, &item.Name, &item.Incoming, &item.Outgoing)
			item.Count = item.Incoming + item.Outgoing
		case "department_load":
			err = rows.Scan(&item.Key, &item.Name, &item.Period, &item.Count)
		case "execution_time":
			err = rows.Scan(&item.Key, &item.Name, &item.Count, &item.Metric)
		case "overdue_rate":
			err = rows.Scan(&item.Key, &item.Name, &item.Count, &item.Overdue, &item.Metric)
		}
		if err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read report: %w", err)
	}
	return result, nil
}

func nullableID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableStringValue(value string) any { return nullableID(value) }

func deadlineReportArgs(req models.ReportRequest, start, end time.Time, zone string) []any {
	return []any{start, end, nullableID(req.DepartmentID), zone, nullableStringValue(req.KindCode), nullableID(req.NomenclatureID), nullableID(req.UserID), nullableStringValue(req.Status)}
}

func (r *ReportingRepository) UniqueDocumentCount(ctx context.Context, start, end time.Time, req models.ReportRequest) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT d.id) FROM documents d
		LEFT JOIN document_correspondent_registrations cr ON cr.document_id=d.id AND d.kind='incoming_letter'
		LEFT JOIN outgoing_document_details od ON od.document_id=d.id AND d.kind='outgoing_letter'
		WHERE d.kind IN ('incoming_letter','outgoing_letter') AND `+r.documentPeriodPredicate(req)+` AND `+r.readPredicate("d")+`
		AND (cr.correspondent_org_id IS NOT NULL OR od.recipient_org_id IS NOT NULL)
		AND ($3::uuid IS NULL OR cr.correspondent_org_id=$3::uuid OR od.recipient_org_id=$3::uuid)
		AND ($4::text IS NULL OR d.kind=$4) AND ($5::uuid IS NULL OR d.nomenclature_id=$5::uuid)
		AND ($6::uuid IS NULL OR d.created_by=$6::uuid)`, start, end, nullableID(req.OrganizationID), nullableStringValue(req.KindCode), nullableID(req.NomenclatureID), nullableID(req.UserID)).Scan(&count)
	return count, err
}

func (r *ReportingRepository) documentPeriodPredicate(req models.ReportRequest) string {
	if req.PeriodBasis == "due_date" || req.PeriodBasis == "executed_on" {
		return `EXISTS(SELECT 1 FROM document_deadline_controls a WHERE a.document_id=d.id AND ` + reportPeriodColumn(req) + ` >= $1::date AND ` + reportPeriodColumn(req) + ` <= $2::date)`
	}
	return `d.registration_date >= $1::date AND d.registration_date <= $2::date`
}
func (r *ReportingRepository) FilterDescription(ctx context.Context, req models.ReportRequest) (string, error) {
	options, err := r.FilterOptions()
	if err != nil {
		return "", err
	}
	people := options.Executors
	if req.Template == "organizations" {
		people = options.Users
	}
	parts := []string{"Область: доступные документы", "Основание периода: " + map[string]string{"registration_date": "дата регистрации документа", "due_date": "срок контроля", "executed_on": "фактическая дата исполнения"}[req.PeriodBasis]}
	for _, filter := range []struct {
		id, label string
		options   []models.StatisticsOption
	}{{req.OrganizationID, "Организация", options.Organizations}, {req.DepartmentID, "Подразделение", options.Departments}, {req.NomenclatureID, "Дело", options.Nomenclature}, {req.UserID, "Сотрудник", people}} {
		if filter.id == "" {
			continue
		}
		found := false
		for _, option := range filter.options {
			if option.Value == filter.id {
				parts = append(parts, filter.label+": "+option.Label)
				found = true
				break
			}
		}
		if !found {
			return "", models.NewBadRequest("недоступное значение фильтра")
		}
	}
	if req.KindCode != "" {
		parts = append(parts, "Вид: "+models.DocumentKind(req.KindCode).Label())
	}
	if req.Status != "" {
		parts = append(parts, "Состояние: "+req.Status)
	}
	return strings.Join(parts, "; "), nil
}

func (r *ReportingRepository) OpenOverdueCount(ctx context.Context, asOf time.Time, req models.ReportRequest) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM document_deadline_controls a JOIN documents d ON d.id=a.document_id JOIN deadline_executors e ON e.id=a.executor_id LEFT JOIN users u ON u.id=e.user_id WHERE a.status='active' AND a.due_date < $1::date AND `+r.readPredicate("d")+` AND ($2::uuid IS NULL OR u.department_id=$2::uuid) AND ($3::text IS NULL OR d.kind=$3) AND ($4::uuid IS NULL OR d.nomenclature_id=$4::uuid) AND ($5::uuid IS NULL OR a.executor_id=$5::uuid) AND ($6::text IS NULL OR a.status=$6)`, asOf.Format(time.DateOnly), nullableID(req.DepartmentID), nullableStringValue(req.KindCode), nullableID(req.NomenclatureID), nullableID(req.UserID), nullableStringValue(req.Status)).Scan(&count)
	return count, err
}

func (r *ReportingRepository) FilterOptions() (*models.ReportFilters, error) {
	stats := NewStatisticsRepository(r.db).ForUser(r.userID)
	nomenclature, err := stats.GetNomenclatureOptions()
	if err != nil {
		return nil, err
	}
	users, err := stats.GetUserOptions()
	if err != nil {
		return nil, err
	}
	kinds := make([]models.StatisticsOption, 0)
	for _, spec := range models.AllDocumentKindSpecs() {
		kinds = append(kinds, models.StatisticsOption{Value: string(spec.Code), Label: spec.Name})
	}
	executors, err := stats.GetExecutorOptions()
	if err != nil {
		return nil, err
	}
	departments, err := r.options(`SELECT DISTINCT dep.id::text,dep.name FROM departments dep JOIN users u ON u.department_id=dep.id JOIN deadline_executors e ON e.user_id=u.id JOIN document_deadline_controls a ON a.executor_id=e.id JOIN documents d ON d.id=a.document_id WHERE ` + r.readPredicate("d") + ` ORDER BY dep.name`)
	if err != nil {
		return nil, err
	}
	organizations, err := r.options(`SELECT DISTINCT o.id::text,o.name FROM organizations o WHERE EXISTS(SELECT 1 FROM documents d LEFT JOIN document_correspondent_registrations cr ON cr.document_id=d.id LEFT JOIN outgoing_document_details od ON od.document_id=d.id WHERE (cr.correspondent_org_id=o.id OR od.recipient_org_id=o.id) AND ` + r.readPredicate("d") + `) ORDER BY o.name`)
	if err != nil {
		return nil, err
	}
	return &models.ReportFilters{Kinds: kinds, Nomenclature: nomenclature, Users: users, Executors: executors, Departments: departments, Organizations: organizations}, nil
}

func (r *ReportingRepository) options(query string) ([]models.StatisticsOption, error) {
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOptions(rows)
}

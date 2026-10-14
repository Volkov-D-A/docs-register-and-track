package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

type ReportingRepository struct{ db *database.DB }

func NewReportingRepository(db *database.DB) *ReportingRepository {
	return &ReportingRepository{db: db}
}

func (r *ReportingRepository) Run(ctx context.Context, req models.ReportRequest, start, end time.Time) ([]models.ReportRow, error) {
	var query string
	var args []any
	zone := req.Timezone
	if zone == "" {
		zone = "UTC"
	}
	switch req.Template {
	case "organizations":
		query = `WITH letters AS (
			SELECT DISTINCT d.id, cr.correspondent_org_id AS organization_id, 'incoming' AS direction
			FROM documents d JOIN document_correspondent_registrations cr ON cr.document_id = d.id
			WHERE d.kind = 'incoming_letter' AND d.registration_date >= $1::date AND d.registration_date <= $2::date
				AND ($4::text IS NULL OR d.kind=$4) AND ($5::uuid IS NULL OR d.nomenclature_id=$5::uuid) AND ($6::uuid IS NULL OR d.created_by=$6::uuid)
			UNION ALL
			SELECT d.id, od.recipient_org_id, 'outgoing'
			FROM documents d JOIN outgoing_document_details od ON od.document_id = d.id
			WHERE d.kind = 'outgoing_letter' AND d.registration_date >= $1::date AND d.registration_date <= $2::date
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
		FROM assignments a JOIN users u ON u.id = a.executor_id
		JOIN documents d ON d.id=a.document_id
		LEFT JOIN departments dep ON dep.id = u.department_id
		WHERE a.type = 'execution' AND (a.created_at AT TIME ZONE $4)::date >= $1::date AND (a.created_at AT TIME ZONE $4)::date <= $2::date
			AND ($3::uuid IS NULL OR dep.id = $3::uuid)
			AND ($5::text IS NULL OR d.kind=$5) AND ($6::uuid IS NULL OR d.nomenclature_id=$6::uuid)
			AND ($7::uuid IS NULL OR a.executor_id=$7::uuid) AND ($8::text IS NULL OR a.status=$8)
			AND (NOT $9::boolean OR a.deadline IS NOT NULL)
		GROUP BY dep.id, dep.name, date_trunc('month', a.created_at AT TIME ZONE $4)
		ORDER BY date_trunc('month', a.created_at AT TIME ZONE $4), dep.name`
		args = assignmentReportArgs(req, start, end, zone)
	case "execution_time":
		query = `SELECT COALESCE(dep.id::text, ''), COALESCE(dep.name, 'Без подразделения'),
			COUNT(*)::int,
			ROUND(AVG(EXTRACT(EPOCH FROM (a.completed_at - a.created_at)) / 86400)::numeric, 1)::double precision
		FROM assignments a JOIN users u ON u.id = a.executor_id
		JOIN documents d ON d.id=a.document_id
		LEFT JOIN departments dep ON dep.id = u.department_id
		WHERE a.type = 'execution' AND (a.completed_at AT TIME ZONE $4)::date >= $1::date AND (a.completed_at AT TIME ZONE $4)::date <= $2::date
			AND ($3::uuid IS NULL OR dep.id = $3::uuid)
			AND ($5::text IS NULL OR d.kind=$5) AND ($6::uuid IS NULL OR d.nomenclature_id=$6::uuid)
			AND ($7::uuid IS NULL OR a.executor_id=$7::uuid) AND ($8::text IS NULL OR a.status=$8)
			AND (NOT $9::boolean OR a.deadline IS NOT NULL)
		GROUP BY dep.id, dep.name ORDER BY dep.name`
		args = assignmentReportArgs(req, start, end, zone)
	case "overdue_rate":
		query = `SELECT COALESCE(dep.id::text, ''), COALESCE(dep.name, 'Без подразделения'),
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE (a.completed_at AT TIME ZONE $4)::date > a.deadline)::int,
			ROUND((100.0 * COUNT(*) FILTER (WHERE (a.completed_at AT TIME ZONE $4)::date > a.deadline) / COUNT(*))::numeric, 1)::double precision
		FROM assignments a JOIN users u ON u.id = a.executor_id
		JOIN documents d ON d.id=a.document_id
		LEFT JOIN departments dep ON dep.id = u.department_id
		WHERE a.type = 'execution' AND a.deadline IS NOT NULL
			AND (a.completed_at AT TIME ZONE $4)::date >= $1::date AND (a.completed_at AT TIME ZONE $4)::date <= $2::date
			AND ($3::uuid IS NULL OR dep.id = $3::uuid)
			AND ($5::text IS NULL OR d.kind=$5) AND ($6::uuid IS NULL OR d.nomenclature_id=$6::uuid)
			AND ($7::uuid IS NULL OR a.executor_id=$7::uuid) AND ($8::text IS NULL OR a.status=$8)
			AND (NOT $9::boolean OR a.deadline IS NOT NULL)
		GROUP BY dep.id, dep.name ORDER BY dep.name`
		args = assignmentReportArgs(req, start, end, zone)
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

func assignmentReportArgs(req models.ReportRequest, start, end time.Time, zone string) []any {
	return []any{start, end, nullableID(req.DepartmentID), zone, nullableStringValue(req.KindCode), nullableID(req.NomenclatureID), nullableID(req.UserID), nullableStringValue(req.Status), req.OnlyWithDeadline}
}

func (r *ReportingRepository) UniqueDocumentCount(ctx context.Context, start, end time.Time, req models.ReportRequest) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT d.id) FROM documents d
		LEFT JOIN document_correspondent_registrations cr ON cr.document_id=d.id AND d.kind='incoming_letter'
		LEFT JOIN outgoing_document_details od ON od.document_id=d.id AND d.kind='outgoing_letter'
		WHERE d.kind IN ('incoming_letter','outgoing_letter') AND d.registration_date >= $1::date AND d.registration_date <= $2::date
		AND (cr.correspondent_org_id IS NOT NULL OR od.recipient_org_id IS NOT NULL)
		AND ($3::uuid IS NULL OR cr.correspondent_org_id=$3::uuid OR od.recipient_org_id=$3::uuid)
		AND ($4::text IS NULL OR d.kind=$4) AND ($5::uuid IS NULL OR d.nomenclature_id=$5::uuid)
		AND ($6::uuid IS NULL OR d.created_by=$6::uuid)`, start, end, nullableID(req.OrganizationID), nullableStringValue(req.KindCode), nullableID(req.NomenclatureID), nullableID(req.UserID)).Scan(&count)
	return count, err
}

func (r *ReportingRepository) FilterDescription(ctx context.Context, req models.ReportRequest) (string, error) {
	parts := make([]string, 0, 7)
	if req.Template == "organizations" {
		if req.OrganizationID == "" {
			parts = append(parts, "Все организации")
		} else {
			var name string
			err := r.db.QueryRowContext(ctx, `SELECT name FROM organizations WHERE id=$1`, req.OrganizationID).Scan(&name)
			if errors.Is(err, sql.ErrNoRows) {
				return "", models.NewBadRequest("организация не найдена")
			}
			if err != nil {
				return "", err
			}
			parts = append(parts, "Организация: "+name)
		}
	} else if req.DepartmentID == "" {
		parts = append(parts, "Все подразделения")
	} else {
		var name string
		err := r.db.QueryRowContext(ctx, `SELECT name FROM departments WHERE id=$1`, req.DepartmentID).Scan(&name)
		if errors.Is(err, sql.ErrNoRows) {
			return "", models.NewBadRequest("подразделение не найдено")
		}
		if err != nil {
			return "", err
		}
		parts = append(parts, "Подразделение: "+name)
	}
	if req.KindCode != "" {
		parts = append(parts, "Вид документа: "+models.DocumentKind(req.KindCode).Label())
	}
	if req.NomenclatureID != "" {
		var name string
		err := r.db.QueryRowContext(ctx, `SELECT CONCAT(index,' - ',name,' (',year,')') FROM nomenclature WHERE id=$1`, req.NomenclatureID).Scan(&name)
		if errors.Is(err, sql.ErrNoRows) {
			return "", models.NewBadRequest("номенклатура не найдена")
		}
		if err != nil {
			return "", err
		}
		parts = append(parts, "Дело: "+name)
	}
	if req.UserID != "" {
		var name string
		err := r.db.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(full_name,''),login) FROM users WHERE id=$1`, req.UserID).Scan(&name)
		if errors.Is(err, sql.ErrNoRows) {
			return "", models.NewBadRequest("пользователь не найден")
		}
		if err != nil {
			return "", err
		}
		parts = append(parts, "Пользователь: "+name)
	}
	if req.Status != "" {
		parts = append(parts, "Статус: "+req.Status)
	}
	if req.OnlyWithDeadline {
		parts = append(parts, "Только со сроком")
	}
	return strings.Join(parts, "; "), nil
}

func (r *ReportingRepository) OpenOverdueCount(ctx context.Context, asOf time.Time, req models.ReportRequest) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assignments a
		JOIN users u ON u.id=a.executor_id
		JOIN documents d ON d.id=a.document_id
		WHERE a.type='execution' AND a.deadline < $1::date AND a.completed_at IS NULL
		AND a.status NOT IN ('completed','finished','cancelled')
		AND ($2::uuid IS NULL OR u.department_id=$2::uuid)
		AND ($3::text IS NULL OR d.kind=$3) AND ($4::uuid IS NULL OR d.nomenclature_id=$4::uuid)
		AND ($5::uuid IS NULL OR a.executor_id=$5::uuid) AND ($6::text IS NULL OR a.status=$6)`, asOf.Format("2006-01-02"), nullableID(req.DepartmentID), nullableStringValue(req.KindCode), nullableID(req.NomenclatureID), nullableID(req.UserID), nullableStringValue(req.Status)).Scan(&count)
	return count, err
}

func (r *ReportingRepository) FilterOptions() (*models.ReportFilters, error) {
	stats := NewStatisticsRepository(r.db)
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
	return &models.ReportFilters{Kinds: kinds, Nomenclature: nomenclature, Users: users}, nil
}

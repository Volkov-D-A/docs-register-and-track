package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const assignmentOverdueCondition = `
	a.deadline IS NOT NULL
	AND (
		(a.completed_at IS NOT NULL AND a.completed_at::date > a.deadline)
		OR
		(a.completed_at IS NULL AND a.status NOT IN ('completed', 'finished', 'cancelled') AND a.deadline < CURRENT_DATE)
	)
`

const assignmentViolationDateExpr = `
	CASE
		WHEN a.deadline IS NOT NULL AND a.completed_at IS NOT NULL AND a.completed_at::date > a.deadline THEN a.completed_at::date
		WHEN a.deadline IS NOT NULL AND a.completed_at IS NULL AND a.status NOT IN ('completed', 'finished', 'cancelled') AND a.deadline < CURRENT_DATE THEN a.deadline
		ELSE NULL
	END
`

// StatisticsRepository предоставляет запросы для раздела статистики.
type StatisticsRepository struct {
	db *database.DB
}

// NewStatisticsRepository создает новый экземпляр StatisticsRepository.
func NewStatisticsRepository(db *database.DB) *StatisticsRepository {
	return &StatisticsRepository{db: db}
}

// GetDocumentTotalByYear возвращает общее количество документов за год по дате регистрации.
func (r *StatisticsRepository) GetDocumentTotalByYear(yearStart, yearEnd time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM documents
		WHERE registration_date >= $1::date AND registration_date < $2::date
	`, yearStart, yearEnd).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get document total by year: %w", err)
	}
	return count, nil
}

// GetMonthlyDocumentCountsByKind возвращает помесячное количество документов по видам.
func (r *StatisticsRepository) GetMonthlyDocumentCountsByKind(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error) {
	rows, err := r.db.Query(`
		SELECT EXTRACT(MONTH FROM registration_date)::int AS month, kind, COUNT(*)
		FROM documents
		WHERE registration_date >= $1::date AND registration_date < $2::date
		GROUP BY month, kind
		ORDER BY month, kind
	`, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly document counts by kind: %w", err)
	}
	defer rows.Close()

	points := make([]models.StatisticsSeriesPoint, 0)
	for rows.Next() {
		var point models.StatisticsSeriesPoint
		if err := rows.Scan(&point.Month, &point.CategoryKey, &point.Value); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

// GetMonthlyDocumentCountsByRegistrar возвращает помесячное количество документов по зарегистрировавшему пользователю.
func (r *StatisticsRepository) GetMonthlyDocumentCountsByRegistrar(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error) {
	rows, err := r.db.Query(`
		SELECT
			EXTRACT(MONTH FROM d.registration_date)::int AS month,
			d.created_by::text,
			COALESCE(NULLIF(u.full_name, ''), u.login, d.created_by::text) AS user_name,
			COUNT(*)
		FROM documents d
		LEFT JOIN users u ON u.id = d.created_by
		WHERE d.registration_date >= $1::date AND d.registration_date < $2::date
		GROUP BY month, d.created_by, u.full_name, u.login
		ORDER BY month, user_name
	`, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly document counts by registrar: %w", err)
	}
	defer rows.Close()

	points := make([]models.StatisticsSeriesPoint, 0)
	for rows.Next() {
		var point models.StatisticsSeriesPoint
		if err := rows.Scan(&point.Month, &point.CategoryKey, &point.CategoryName, &point.Value); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

// GetDocumentReport возвращает отчет по документам за период с выбранной группировкой.
func (r *StatisticsRepository) GetDocumentReport(startDate, endDate time.Time, groupBy, kindCode, nomenclatureID, userID string) ([]models.StatisticsReportRow, error) {
	selectExpr := ""
	joinClause := ""
	groupExpr := ""
	orderExpr := "name"

	switch groupBy {
	case "kind":
		selectExpr = "d.kind::text AS key, d.kind::text AS name"
		groupExpr = "d.kind"
		orderExpr = "d.kind"
	case "nomenclature":
		selectExpr = "d.nomenclature_id::text AS key, CONCAT(n.index, ' - ', n.name, ' (', n.year, ')') AS name"
		joinClause = "JOIN nomenclature n ON n.id = d.nomenclature_id"
		groupExpr = "d.nomenclature_id, n.index, n.name, n.year"
	case "user":
		selectExpr = "d.created_by::text AS key, COALESCE(NULLIF(u.full_name, ''), u.login, d.created_by::text) AS name"
		joinClause = "LEFT JOIN users u ON u.id = d.created_by"
		groupExpr = "d.created_by, u.full_name, u.login"
	default:
		return nil, fmt.Errorf("unsupported document report group: %s", groupBy)
	}

	where := []string{"d.registration_date >= $1::date", "d.registration_date <= $2::date"}
	args := []any{startDate, endDate}
	argIdx := 3

	if kindCode != "" {
		where = append(where, fmt.Sprintf("d.kind = $%d", argIdx))
		args = append(args, kindCode)
		argIdx++
	}
	if nomenclatureID != "" {
		where = append(where, fmt.Sprintf("d.nomenclature_id = $%d::uuid", argIdx))
		args = append(args, nomenclatureID)
		argIdx++
	}
	if userID != "" {
		where = append(where, fmt.Sprintf("d.created_by = $%d::uuid", argIdx))
		args = append(args, userID)
	}

	query := fmt.Sprintf(`
		SELECT %s, COUNT(*) AS count
		FROM documents d
		%s
		WHERE %s
		GROUP BY %s
		ORDER BY %s
	`, selectExpr, joinClause, strings.Join(where, " AND "), groupExpr, orderExpr)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get document report: %w", err)
	}
	defer rows.Close()

	return scanReportRows(rows)
}

// GetNomenclatureOptions возвращает список номенклатур для фильтров статистики.
func (r *StatisticsRepository) GetNomenclatureOptions() ([]models.StatisticsOption, error) {
	rows, err := r.db.Query(`
		SELECT id::text, CONCAT(index, ' - ', name, ' (', year, ')')
		FROM nomenclature
		ORDER BY year DESC, kind_code, index, name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get nomenclature options: %w", err)
	}
	defer rows.Close()

	return scanOptions(rows)
}

// GetUserOptions возвращает список пользователей для фильтров статистики.
func (r *StatisticsRepository) GetUserOptions() ([]models.StatisticsOption, error) {
	rows, err := r.db.Query(`
		SELECT id::text, COALESCE(NULLIF(full_name, ''), login, id::text)
		FROM users
		ORDER BY full_name, login
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get user options: %w", err)
	}
	defer rows.Close()

	return scanOptions(rows)
}

// GetAssignmentMonthlyOverview возвращает помесячные общие и просроченные поручения.
func (r *StatisticsRepository) GetAssignmentMonthlyOverview(yearStart, yearEnd time.Time) ([]models.AssignmentMonthlyPoint, error) {
	rows, err := r.db.Query(fmt.Sprintf(`
		WITH totals AS (
			SELECT EXTRACT(MONTH FROM created_at)::int AS month, COUNT(*) AS total
			FROM assignments
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY month
		),
		overdue AS (
			SELECT EXTRACT(MONTH FROM violation_date)::int AS month, COUNT(*) AS overdue
			FROM (
				SELECT %s AS violation_date
				FROM assignments a
			) v
			WHERE violation_date >= $1::date AND violation_date < $2::date
			GROUP BY month
		)
		SELECT m.month, COALESCE(t.total, 0), COALESCE(o.overdue, 0)
		FROM generate_series(1, 12) AS m(month)
		LEFT JOIN totals t ON t.month = m.month
		LEFT JOIN overdue o ON o.month = m.month
		ORDER BY m.month
	`, assignmentViolationDateExpr), yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment monthly overview: %w", err)
	}
	defer rows.Close()

	points := make([]models.AssignmentMonthlyPoint, 0, 12)
	for rows.Next() {
		var point models.AssignmentMonthlyPoint
		if err := rows.Scan(&point.Month, &point.Total, &point.Overdue); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

// GetAssignmentMonthlyByExecutor возвращает помесячное количество поручений по основным исполнителям.
func (r *StatisticsRepository) GetAssignmentMonthlyByExecutor(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error) {
	rows, err := r.db.Query(`
		SELECT
			EXTRACT(MONTH FROM a.created_at)::int AS month,
			a.executor_id::text,
			COALESCE(NULLIF(u.full_name, ''), u.login, a.executor_id::text) AS user_name,
			COUNT(*)
		FROM assignments a
		LEFT JOIN users u ON u.id = a.executor_id
		WHERE a.created_at >= $1 AND a.created_at < $2
		GROUP BY month, a.executor_id, u.full_name, u.login
		ORDER BY month, user_name
	`, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment monthly by executor: %w", err)
	}
	defer rows.Close()

	points := make([]models.StatisticsSeriesPoint, 0)
	for rows.Next() {
		var point models.StatisticsSeriesPoint
		if err := rows.Scan(&point.Month, &point.CategoryKey, &point.CategoryName, &point.Value); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

// GetAssignmentOverdueRating возвращает рейтинг основных исполнителей по нарушениям сроков.
func (r *StatisticsRepository) GetAssignmentOverdueRating(yearStart, yearEnd time.Time) ([]models.StatisticsReportRow, error) {
	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT
			a.executor_id::text AS key,
			COALESCE(NULLIF(u.full_name, ''), u.login, a.executor_id::text) AS name,
			COUNT(*) AS count
		FROM assignments a
		LEFT JOIN users u ON u.id = a.executor_id
		WHERE %s
		  AND (%s) >= $1::date
		  AND (%s) < $2::date
		GROUP BY a.executor_id, u.full_name, u.login
		ORDER BY count DESC, name
	`, assignmentOverdueCondition, assignmentViolationDateExpr, assignmentViolationDateExpr), yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment overdue rating: %w", err)
	}
	defer rows.Close()

	return scanReportRows(rows)
}

// GetAssignmentStatusCounts возвращает количество поручений по статусам.
func (r *StatisticsRepository) GetAssignmentStatusCounts() ([]models.StatisticsReportRow, error) {
	rows, err := r.db.Query(`
		SELECT status AS key, status AS name, COUNT(*) AS count
		FROM assignments
		GROUP BY status
		ORDER BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment status counts: %w", err)
	}
	defer rows.Close()

	return scanReportRows(rows)
}

// GetAssignmentReport возвращает отчет по поручениям за период, сгруппированный по основным исполнителям.
func (r *StatisticsRepository) GetAssignmentReport(startDate, endDate time.Time, onlyOverdue bool, userID string) ([]models.StatisticsReportRow, error) {
	where := []string{}
	args := []any{startDate, endDate}
	argIdx := 3

	if onlyOverdue {
		where = append(where,
			assignmentOverdueCondition,
			fmt.Sprintf("(%s) >= $1::date", assignmentViolationDateExpr),
			fmt.Sprintf("(%s) <= $2::date", assignmentViolationDateExpr),
		)
	} else {
		where = append(where, "a.created_at::date >= $1::date", "a.created_at::date <= $2::date")
	}

	if userID != "" {
		where = append(where, fmt.Sprintf("a.executor_id = $%d::uuid", argIdx))
		args = append(args, userID)
	}

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT
			a.executor_id::text AS key,
			COALESCE(NULLIF(u.full_name, ''), u.login, a.executor_id::text) AS name,
			COUNT(*) AS count
		FROM assignments a
		LEFT JOIN users u ON u.id = a.executor_id
		WHERE %s
		GROUP BY a.executor_id, u.full_name, u.login
		ORDER BY count DESC, name
	`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment report: %w", err)
	}
	defer rows.Close()

	return scanReportRows(rows)
}

// GetSystemUserCount возвращает общее количество пользователей.
func (r *StatisticsRepository) GetSystemUserCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get system user count: %w", err)
	}
	return count, nil
}

// GetSystemDocumentCount возвращает общее количество документов.
func (r *StatisticsRepository) GetSystemDocumentCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get system document count: %w", err)
	}
	return count, nil
}

// GetDBSize возвращает размер базы данных в человекочитаемом формате.
func (r *StatisticsRepository) GetDBSize() string {
	var size string
	err := r.db.QueryRow("SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&size)
	if err != nil {
		return "N/A"
	}
	return size
}

func scanOptions(rows *sql.Rows) ([]models.StatisticsOption, error) {
	options := make([]models.StatisticsOption, 0)
	for rows.Next() {
		var option models.StatisticsOption
		if err := rows.Scan(&option.Value, &option.Label); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return options, nil
}

func scanReportRows(rows *sql.Rows) ([]models.StatisticsReportRow, error) {
	result := make([]models.StatisticsReportRow, 0)
	for rows.Next() {
		var row models.StatisticsReportRow
		if err := rows.Scan(&row.Key, &row.Name, &row.Count); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func setupStatisticsRepository(t *testing.T) (*StatisticsRepository, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewStatisticsRepository(&database.DB{DB: db}), mock, func() { db.Close() }
}

func TestStatisticsRepository_GetDocumentTotalByYear(t *testing.T) {
	repo, mock, cleanup := setupStatisticsRepository(t)
	defer cleanup()

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM documents`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.GetDocumentTotalByYear(start, end)

	require.NoError(t, err)
	assert.Equal(t, 42, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStatisticsRepository_MonthlyDocumentCounts(t *testing.T) {
	repo, mock, cleanup := setupStatisticsRepository(t)
	defer cleanup()

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	t.Run("by kind", func(t *testing.T) {
		mock.ExpectQuery(`EXTRACT\(MONTH FROM registration_date\)::int AS month, kind, COUNT\(\*\)`).
			WithArgs(start, end).
			WillReturnRows(sqlmock.NewRows([]string{"month", "kind", "count"}).
				AddRow(1, string(models.DocumentKindIncomingLetter), 3).
				AddRow(2, string(models.DocumentKindOutgoingLetter), 4))

		points, err := repo.GetMonthlyDocumentCountsByKind(start, end)

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsSeriesPoint{
			{Month: 1, CategoryKey: string(models.DocumentKindIncomingLetter), Value: 3},
			{Month: 2, CategoryKey: string(models.DocumentKindOutgoingLetter), Value: 4},
		}, points)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("by registrar", func(t *testing.T) {
		mock.ExpectQuery(`EXTRACT\(MONTH FROM d\.registration_date\)::int AS month`).
			WithArgs(start, end).
			WillReturnRows(sqlmock.NewRows([]string{"month", "created_by", "user_name", "count"}).
				AddRow(3, "user-1", "Регистратор", 5))

		points, err := repo.GetMonthlyDocumentCountsByRegistrar(start, end)

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsSeriesPoint{
			{Month: 3, CategoryKey: "user-1", CategoryName: "Регистратор", Value: 5},
		}, points)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStatisticsRepository_GetDocumentReport(t *testing.T) {
	repo, mock, cleanup := setupStatisticsRepository(t)
	defer cleanup()

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
	nomenclatureID := "7d402d95-5353-4a7c-a8b9-0f9172f20a0a"
	userID := "34d2412c-3784-4e6a-9279-5b8bccf4a7e0"

	t.Run("kind with filters", func(t *testing.T) {
		mock.ExpectQuery(`SELECT d\.kind::text AS key, d\.kind::text AS name, COUNT\(\*\) AS count`).
			WithArgs(start, end, string(models.DocumentKindIncomingLetter), nomenclatureID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"key", "name", "count"}).
				AddRow(string(models.DocumentKindIncomingLetter), string(models.DocumentKindIncomingLetter), 6))

		rows, err := repo.GetDocumentReport(start, end, "kind", string(models.DocumentKindIncomingLetter), nomenclatureID, userID)

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsReportRow{{
			Key:   string(models.DocumentKindIncomingLetter),
			Name:  string(models.DocumentKindIncomingLetter),
			Count: 6,
		}}, rows)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nomenclature", func(t *testing.T) {
		mock.ExpectQuery(`JOIN nomenclature n ON n\.id = d\.nomenclature_id`).
			WithArgs(start, end).
			WillReturnRows(sqlmock.NewRows([]string{"key", "name", "count"}).
				AddRow(nomenclatureID, "01-01 - Дело (2026)", 7))

		rows, err := repo.GetDocumentReport(start, end, "nomenclature", "", "", "")

		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "01-01 - Дело (2026)", rows[0].Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("unsupported group", func(t *testing.T) {
		rows, err := repo.GetDocumentReport(start, end, "unknown", "", "", "")

		require.Error(t, err)
		assert.Nil(t, rows)
		assert.Contains(t, err.Error(), "unsupported document report group")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStatisticsRepository_Options(t *testing.T) {
	repo, mock, cleanup := setupStatisticsRepository(t)
	defer cleanup()

	t.Run("nomenclature", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id::text, CONCAT\(index, ' - ', name, ' \(', year, '\)'\)\s+FROM nomenclature`).
			WillReturnRows(sqlmock.NewRows([]string{"value", "label"}).
				AddRow("nom-1", "01-01 - Дело (2026)"))

		options, err := repo.GetNomenclatureOptions()

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsOption{{Value: "nom-1", Label: "01-01 - Дело (2026)"}}, options)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("users", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id::text, COALESCE\(NULLIF\(full_name, ''\), login, id::text\)\s+FROM users`).
			WillReturnRows(sqlmock.NewRows([]string{"value", "label"}).
				AddRow("user-1", "Иван Иванов"))

		options, err := repo.GetUserOptions()

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsOption{{Value: "user-1", Label: "Иван Иванов"}}, options)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStatisticsRepository_AssignmentStatistics(t *testing.T) {
	repo, mock, cleanup := setupStatisticsRepository(t)
	defer cleanup()

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	t.Run("monthly overview", func(t *testing.T) {
		mock.ExpectQuery(`WITH totals AS`).
			WithArgs(start, end).
			WillReturnRows(sqlmock.NewRows([]string{"month", "total", "overdue"}).
				AddRow(1, 10, 2).
				AddRow(2, 4, 1))

		points, err := repo.GetAssignmentMonthlyOverview(start, end)

		require.NoError(t, err)
		assert.Equal(t, []models.AssignmentMonthlyPoint{
			{Month: 1, Total: 10, Overdue: 2},
			{Month: 2, Total: 4, Overdue: 1},
		}, points)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("monthly by executor", func(t *testing.T) {
		mock.ExpectQuery(`EXTRACT\(MONTH FROM a\.created_at\)::int AS month`).
			WithArgs(start, end).
			WillReturnRows(sqlmock.NewRows([]string{"month", "executor_id", "user_name", "count"}).
				AddRow(3, "user-1", "Исполнитель", 8))

		points, err := repo.GetAssignmentMonthlyByExecutor(start, end)

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsSeriesPoint{{
			Month: 3, CategoryKey: "user-1", CategoryName: "Исполнитель", Value: 8,
		}}, points)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("overdue rating", func(t *testing.T) {
		mock.ExpectQuery(`a\.executor_id::text AS key`).
			WithArgs(start, end).
			WillReturnRows(sqlmock.NewRows([]string{"key", "name", "count"}).
				AddRow("user-1", "Исполнитель", 2))

		rows, err := repo.GetAssignmentOverdueRating(start, end)

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsReportRow{{Key: "user-1", Name: "Исполнитель", Count: 2}}, rows)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("status counts", func(t *testing.T) {
		mock.ExpectQuery(`SELECT status AS key, status AS name, COUNT\(\*\) AS count\s+FROM assignments`).
			WillReturnRows(sqlmock.NewRows([]string{"key", "name", "count"}).
				AddRow("completed", "completed", 5))

		rows, err := repo.GetAssignmentStatusCounts()

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsReportRow{{Key: "completed", Name: "completed", Count: 5}}, rows)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("report only overdue with user filter", func(t *testing.T) {
		userID := "34d2412c-3784-4e6a-9279-5b8bccf4a7e0"
		mock.ExpectQuery(`WHERE\s+.*a\.deadline IS NOT NULL.*a\.executor_id = \$3::uuid`).
			WithArgs(start, end, userID).
			WillReturnRows(sqlmock.NewRows([]string{"key", "name", "count"}).
				AddRow(userID, "Исполнитель", 3))

		rows, err := repo.GetAssignmentReport(start, end, true, userID)

		require.NoError(t, err)
		assert.Equal(t, []models.StatisticsReportRow{{Key: userID, Name: "Исполнитель", Count: 3}}, rows)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStatisticsRepository_SystemCountsAndDBSize(t *testing.T) {
	repo, mock, cleanup := setupStatisticsRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(9))
	userCount, err := repo.GetSystemUserCount()
	require.NoError(t, err)
	assert.Equal(t, 9, userCount)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(14))
	documentCount, err := repo.GetSystemDocumentCount()
	require.NoError(t, err)
	assert.Equal(t, 14, documentCount)

	mock.ExpectQuery(`SELECT pg_size_pretty\(pg_database_size\(current_database\(\)\)\)`).
		WillReturnRows(sqlmock.NewRows([]string{"size"}).AddRow("128 MB"))
	assert.Equal(t, "128 MB", repo.GetDBSize())

	mock.ExpectQuery(`SELECT pg_size_pretty\(pg_database_size\(current_database\(\)\)\)`).
		WillReturnError(sql.ErrConnDone)
	assert.Equal(t, "N/A", repo.GetDBSize())

	require.NoError(t, mock.ExpectationsWereMet())
}

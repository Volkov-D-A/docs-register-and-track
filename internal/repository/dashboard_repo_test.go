package repository

import (
	"testing"
	"time"

	"docflow/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardRepository_GetExecutorStatusCounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})
	userID := uuid.New()

	query := `SELECT COUNT\(\*\) FILTER \(WHERE status = 'new'\), COUNT\(\*\) FILTER \(WHERE status = 'in_progress'\) FROM assignments a WHERE executor_id = \$1 OR EXISTS \(SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = \$1\)`

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(
		sqlmock.NewRows([]string{"new_count", "in_progress_count"}).AddRow(5, 3),
	)

	newCount, inProgressCount, err := repo.GetExecutorStatusCounts(userID)
	require.NoError(t, err)
	assert.Equal(t, 5, newCount)
	assert.Equal(t, 3, inProgressCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetExecutorOverdueCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})
	userID := uuid.New()

	query := `SELECT COUNT\(\*\) FROM assignments a WHERE \(executor_id = \$1 OR EXISTS \(SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = \$1\)\) AND \( \(status IN \('new', 'in_progress'\) AND deadline < CURRENT_DATE\) OR \(status = 'completed' AND completed_at::date > deadline\) \)`

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(2),
	)

	count, err := repo.GetExecutorOverdueCount(userID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetExecutorFinishedCounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})
	userID := uuid.New()

	query := `SELECT COUNT\(\*\) FILTER \(WHERE status = 'finished'\), COUNT\(\*\) FILTER \(WHERE status = 'finished' AND completed_at::date > deadline\) FROM assignments a WHERE \(executor_id = \$1 OR EXISTS \(SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = \$1\)\)`

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(
		sqlmock.NewRows([]string{"finished", "finished_late"}).AddRow(10, 2),
	)

	finished, finishedLate, err := repo.GetExecutorFinishedCounts(userID)
	require.NoError(t, err)
	assert.Equal(t, 10, finished)
	assert.Equal(t, 2, finishedLate)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetExpiringAssignments(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})
	userID := uuid.New()
	now := time.Now()

	query := `SELECT a.id, a.content, a.deadline, a.status, a.document_id, a.document_type, u.full_name as executor_name, COALESCE\(inc.incoming_number, out.outgoing_number\) as doc_number FROM assignments a LEFT JOIN users u ON a.executor_id = u.id LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming' LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing' WHERE \(a.executor_id = \$1 OR EXISTS \(SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = \$1\)\) AND a.status IN \('new', 'in_progress'\) AND a.deadline BETWEEN CURRENT_DATE AND \(CURRENT_DATE \+ INTERVAL '1 day' \* \$2\) ORDER BY a.deadline ASC`

	rows := sqlmock.NewRows([]string{
		"id", "content", "deadline", "status", "document_id", "document_type", "executor_name", "doc_number",
	}).AddRow(uuid.New(), "Content", now, "new", uuid.New(), "incoming", "Executor", "doc-1")

	mock.ExpectQuery(query).WithArgs(userID, 3).WillReturnRows(rows)

	res, err := repo.GetExpiringAssignments(&userID, 3)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetDocCountsByPeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	query := `SELECT 
		\(SELECT COUNT\(\*\) FROM incoming_documents WHERE created_at BETWEEN \$1 AND \$2\),
		\(SELECT COUNT\(\*\) FROM outgoing_documents WHERE created_at BETWEEN \$1 AND \$2\)`
		
	mock.ExpectQuery(query).WithArgs(start, end).WillReturnRows(sqlmock.NewRows([]string{"incoming", "outgoing"}).AddRow(15, 7))

	inc, out, err := repo.GetDocCountsByPeriod(start, end)
	require.NoError(t, err)
	assert.Equal(t, 15, inc)
	assert.Equal(t, 7, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetOverdueCountByPeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	query := `SELECT COUNT\(\*\) FROM assignments WHERE deadline BETWEEN \$1 AND \$2 AND \( \(status IN \('new', 'in_progress'\) AND deadline < CURRENT_DATE\) OR \(status = 'completed' AND completed_at::date > deadline\) \)`

	mock.ExpectQuery(query).WithArgs(start, end).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	count, err := repo.GetOverdueCountByPeriod(start, end)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetFinishedCountsByPeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	query := `SELECT COUNT\(\*\) FILTER \(WHERE status = 'finished'\), COUNT\(\*\) FILTER \(WHERE status = 'finished' AND COALESCE\(completed_at, updated_at\)::date > deadline\) FROM assignments WHERE status = 'finished' AND COALESCE\(completed_at, updated_at\) BETWEEN \$1 AND \$2`

	mock.ExpectQuery(query).WithArgs(start, end).WillReturnRows(
		sqlmock.NewRows([]string{"finished", "finished_late"}).AddRow(20, 5),
	)

	finished, late, err := repo.GetFinishedCountsByPeriod(start, end)
	require.NoError(t, err)
	assert.Equal(t, 20, finished)
	assert.Equal(t, 5, late)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetAdminUserCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.GetAdminUserCount()
	require.NoError(t, err)
	assert.Equal(t, 42, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetAdminDocCounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})

	query := `SELECT 
		\(SELECT COUNT\(\*\) FROM incoming_documents\),
		\(SELECT COUNT\(\*\) FROM outgoing_documents\)`
		
	mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{"incoming", "outgoing"}).AddRow(100, 50))

	inc, out, err := repo.GetAdminDocCounts()
	require.NoError(t, err)
	assert.Equal(t, 100, inc)
	assert.Equal(t, 50, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_GetDBSize(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})

	query := `SELECT pg_size_pretty\(pg_database_size\(current_database\(\)\)\)`
	mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{"size"}).AddRow("15 MB"))

	size := repo.GetDBSize()
	assert.Equal(t, "15 MB", size)
	require.NoError(t, mock.ExpectationsWereMet())
}

package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func setupAssignmentSeriesRepo(t *testing.T) (*AssignmentRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	wrapper := &database.DB{DB: db}
	repo := NewAssignmentRepository(wrapper)
	repo.SetOutbox(NewOutboxRepository(wrapper))
	return repo, mockDB, func() { _ = db.Close() }
}

func requireRepositoryConflict(t *testing.T, err error) {
	t.Helper()
	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, "CONFLICT", appErr.Kind)
	assert.Equal(t, 409, appErr.Code)
}

func TestAssignmentSeriesRepositoryCreatesTemplateAndFirstIteration(t *testing.T) {
	repo, mockDB, closeDB := setupAssignmentSeriesRepo(t)
	defer closeDB()
	seriesID, assignmentID, documentID, executorID, actorID, coExecutorID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	deadline := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()

	mockDB.ExpectBegin()
	mockDB.ExpectExec(`INSERT INTO assignment_series`).WithArgs(seriesID, documentID, executorID, "Отчёт", "month", 3, "last_day", nil, actorID).WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectExec(`INSERT INTO assignments`).WithArgs(assignmentID, documentID, executorID, "Отчёт", deadline, seriesID).WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectExec(`INSERT INTO assignment_series_co_executors`).WithArgs(seriesID, coExecutorID).WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectExec(`INSERT INTO assignment_co_executors`).WithArgs(assignmentID, coExecutorID).WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectExec(`UPDATE assignment_series SET current_assignment_id=\$1`).WithArgs(assignmentID, seriesID).WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectCommit()
	mockDB.ExpectQuery(`SELECT s.id,s.document_id`).WithArgs(seriesID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "document_id", "kind", "registration_number", "executor_id", "full_name", "content",
		"interval_unit", "interval_value", "day_rule", "day_of_month", "current_assignment_id",
		"current_iteration", "active", "created_by", "cancelled_by", "cancelled_at", "created_at", "updated_at",
	}).AddRow(seriesID, documentID, "incoming_letter", "ВХ-1", executorID, "Исполнитель", "Отчёт", "month", 3, "last_day", nil, assignmentID, 1, true, actorID, nil, nil, now, now))
	mockDB.ExpectQuery(`SELECT u.id,u.login,u.full_name`).WithArgs(seriesID).WillReturnRows(sqlmock.NewRows([]string{"id", "login", "full_name"}))

	series, err := repo.CreateSeriesWithFirstAssignment(seriesID, assignmentID, documentID, executorID, actorID, "Отчёт", deadline, "month", 3, "last_day", 0, []string{coExecutorID.String()}, nil)
	require.NoError(t, err)
	require.NotNil(t, series)
	require.NotNil(t, series.CurrentAssignmentID)
	assert.Equal(t, assignmentID, *series.CurrentAssignmentID)
	assert.Equal(t, 1, series.CurrentIteration)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestAssignmentSeriesRepositoryFindByNonSeriesAssignment(t *testing.T) {
	repo, mockDB, closeDB := setupAssignmentSeriesRepo(t)
	defer closeDB()
	assignmentID := uuid.New()
	mockDB.ExpectQuery(`SELECT series_id FROM assignments`).WithArgs(assignmentID).WillReturnRows(sqlmock.NewRows([]string{"series_id"}).AddRow(nil))

	series, err := repo.GetAssignmentSeriesByAssignment(assignmentID)
	require.NoError(t, err)
	require.Nil(t, series)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestAssignmentSeriesRepositoryLoadsHistoryAndCoExecutors(t *testing.T) {
	repo, mockDB, closeDB := setupAssignmentSeriesRepo(t)
	defer closeDB()
	seriesID, assignmentID, documentID, executorID, coExecutorID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	deadline, now := time.Now().UTC().AddDate(0, 1, 0), time.Now().UTC()
	mockDB.ExpectQuery(`SELECT a.id,a.document_id`).WithArgs(seriesID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "document_id", "kind", "executor_id", "executor_name", "content", "deadline", "status", "report", "completed_at",
		"series_id", "iteration_number", "planned_deadline", "is_series_current", "created_at", "updated_at", "registration_number", "document_content",
	}).AddRow(assignmentID, documentID, "incoming_letter", executorID, "Исполнитель", "Отчёт", deadline, "finished", "готово", now, seriesID, 1, deadline, false, now, now, "ВХ-1", "Тема"))
	mockDB.ExpectQuery(`SELECT ce.assignment_id,u.id,u.login,u.full_name`).WithArgs(sqlmock.AnyArg()).WillReturnRows(
		sqlmock.NewRows([]string{"assignment_id", "user_id", "login", "full_name"}).AddRow(assignmentID, coExecutorID, "co", "Соисполнитель"),
	)

	history, err := repo.GetAssignmentSeriesHistory(seriesID)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, assignmentID, history[0].ID)
	assert.Equal(t, "готово", history[0].Report)
	require.Len(t, history[0].CoExecutorIDs, 1)
	assert.Equal(t, coExecutorID.String(), history[0].CoExecutorIDs[0])
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestAssignmentSeriesRepositoryRejectsInactiveMutation(t *testing.T) {
	t.Run("update cancelled series", func(t *testing.T) {
		repo, mockDB, closeDB := setupAssignmentSeriesRepo(t)
		defer closeDB()
		seriesID, executorID := uuid.New(), uuid.New()
		mockDB.ExpectBegin()
		mockDB.ExpectExec(`UPDATE assignment_series SET executor_id=\$1`).WithArgs(executorID, "Отчёт", "month", 1, "last_day", nil, seriesID).WillReturnResult(sqlmock.NewResult(0, 0))
		mockDB.ExpectRollback()

		series, err := repo.UpdateAssignmentSeries(seriesID, executorID, "Отчёт", "month", 1, "last_day", 0, nil, nil)
		require.Nil(t, series)
		requireRepositoryConflict(t, err)
		require.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("cancel already cancelled series", func(t *testing.T) {
		repo, mockDB, closeDB := setupAssignmentSeriesRepo(t)
		defer closeDB()
		seriesID, actorID := uuid.New(), uuid.New()
		mockDB.ExpectBegin()
		mockDB.ExpectExec(`UPDATE assignment_series SET active=FALSE`).WithArgs(actorID, seriesID).WillReturnResult(sqlmock.NewResult(0, 0))
		mockDB.ExpectRollback()

		requireRepositoryConflict(t, repo.CancelAssignmentSeries(seriesID, actorID, nil))
		require.NoError(t, mockDB.ExpectationsWereMet())
	})
}

func TestAssignmentSeriesRepositoryRejectsStaleAdvance(t *testing.T) {
	repo, mockDB, closeDB := setupAssignmentSeriesRepo(t)
	defer closeDB()
	seriesID, currentID := uuid.New(), uuid.New()
	expectedRevision := time.Now().UTC().Add(-time.Minute)
	actualRevision := time.Now().UTC()
	mockDB.ExpectBegin()
	mockDB.ExpectQuery(`SELECT active,current_assignment_id,updated_at FROM assignment_series`).WithArgs(seriesID).WillReturnRows(
		sqlmock.NewRows([]string{"active", "current_assignment_id", "updated_at"}).AddRow(true, currentID, actualRevision),
	)
	mockDB.ExpectRollback()

	assignment, err := repo.FinishSeriesIterationWithNext(currentID, seriesID, uuid.New(), expectedRevision, "готово", nil, time.Now(), 2, uuid.New(), "Отчёт", nil, nil, nil)
	require.Nil(t, assignment)
	requireRepositoryConflict(t, err)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestAssignmentSeriesRepositoryReturnsNilForMissingSeries(t *testing.T) {
	repo, mockDB, closeDB := setupAssignmentSeriesRepo(t)
	defer closeDB()
	seriesID := uuid.New()
	mockDB.ExpectQuery(`SELECT s.id,s.document_id`).WithArgs(seriesID).WillReturnError(sql.ErrNoRows)

	series, err := repo.GetAssignmentSeries(seriesID)
	require.NoError(t, err)
	require.Nil(t, series)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

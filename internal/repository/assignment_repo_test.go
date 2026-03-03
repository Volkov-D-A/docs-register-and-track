package repository

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"docflow/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignmentRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAssignmentRepository(&database.DB{DB: db})
	assignID := uuid.New()
	now := time.Now()

	expectedQuery := `SELECT
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
		WHERE a.id = $1`

	t.Run("success without co-executors", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "document_id", "document_type",
			"executor_id", "full_name",
			"content", "deadline", "status", "report", "completed_at",
			"created_at", "updated_at",
			"doc_number", "doc_subject",
		}).AddRow(
			assignID, uuid.New(), "incoming",
			uuid.New(), "Иванов И.И.",
			"Выполнить задачу", now, "new", nil, nil,
			now, now,
			"ВХ-1", "Тема",
		)

		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(assignID).WillReturnRows(rows)

		// Запрос соисполнителей (пустой)
		coExecQuery := `SELECT u.id, u.login, u.full_name
		FROM assignment_co_executors ce
		JOIN users u ON ce.user_id = u.id
		WHERE ce.assignment_id = $1`
		mock.ExpectQuery(regexp.QuoteMeta(coExecQuery)).WithArgs(assignID).WillReturnRows(sqlmock.NewRows([]string{"id", "login", "full_name"}))

		assign, err := repo.GetByID(assignID)
		require.NoError(t, err)
		require.NotNil(t, assign)
		assert.Equal(t, assignID, assign.ID)
		assert.Equal(t, "Выполнить задачу", assign.Content)
		assert.Equal(t, "new", assign.Status)
		assert.Empty(t, assign.CoExecutors)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(assignID).WillReturnError(sql.ErrNoRows)

		assign, err := repo.GetByID(assignID)
		require.NoError(t, err)
		require.Nil(t, assign)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAssignmentRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAssignmentRepository(&database.DB{DB: db})
	assignID := uuid.New()

	mock.ExpectExec(`DELETE FROM assignments WHERE id = \$1`).WithArgs(assignID).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(assignID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAssignmentRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAssignmentRepository(&database.DB{DB: db})
	assignID := uuid.New()
	now := time.Now()
	docID := uuid.New()
	execID := uuid.New()
	coExecID := uuid.New()

	mock.ExpectBegin()

	insertQuery := `INSERT INTO assignments (
			document_id, document_type, executor_id,
			content, deadline, status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).WithArgs(
		docID, "incoming", execID, "Текст", &now, "new",
	).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(assignID, now, now))

	mock.ExpectPrepare(`INSERT INTO assignment_co_executors \(assignment_id, user_id\) VALUES \(\$1, \$2\)`).
		ExpectExec().WithArgs(assignID, coExecID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	// После Commit идет GetByID
	expectedGetQuery := `SELECT
			a.id, a.document_id, a.document_type,
			a.executor_id, u_executor.full_name,
			a.content, a.deadline, a.status, a.report, a.completed_at,
			a.created_at, a.updated_at,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number,
			COALESCE(inc.subject, out.subject) as doc_subject
		FROM assignments a`

	rows := sqlmock.NewRows([]string{
		"id", "document_id", "document_type",
		"executor_id", "full_name",
		"content", "deadline", "status", "report", "completed_at",
		"created_at", "updated_at",
		"doc_number", "doc_subject",
	}).AddRow(assignID, docID, "incoming", execID, "Иванов", "Текст", now, "new", nil, nil, now, now, "", "")

	mock.ExpectQuery(regexp.QuoteMeta(expectedGetQuery)).WithArgs(assignID).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT u.id, u.login, u.full_name FROM assignment_co_executors`).WithArgs(assignID).WillReturnRows(sqlmock.NewRows([]string{"id", "login", "full_name"}))

	assign, err := repo.Create(docID, "incoming", execID, "Текст", &now, []string{coExecID.String()})
	require.NoError(t, err)
	require.NotNil(t, assign)
	assert.Equal(t, assignID, assign.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

package repository

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignmentRepository_GetByID(t *testing.T) {
	// Получение деталей поручения по его ID
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
			COALESCE(inc.content, out.subject) as doc_subject
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
	// Удаление поручения по его ID
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
	// Создание нового поручения с привязкой соисполнителей
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
			COALESCE(inc.content, out.subject) as doc_subject
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

func TestAssignmentRepository_Update(t *testing.T) {
	// Обновление существующего поручения и списка соисполнителей
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAssignmentRepository(&database.DB{DB: db})
	assignID := uuid.New()
	execID := uuid.New()
	coExecID := uuid.New()
	now := time.Now()

	mock.ExpectBegin()

	updateQuery := `UPDATE assignments SET executor_id = \$1, content = \$2, deadline = \$3, status = \$4, report = \$5, completed_at = \$6, updated_at = NOW\(\) WHERE id = \$7`
	mock.ExpectExec(updateQuery).WithArgs(execID, "Обновленный текст", &now, "in_progress", "Отчет", &now, assignID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`DELETE FROM assignment_co_executors WHERE assignment_id = \$1`).WithArgs(assignID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectPrepare(`INSERT INTO assignment_co_executors \(assignment_id, user_id\) VALUES \(\$1, \$2\)`).
		ExpectExec().WithArgs(assignID, coExecID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	// getByID call mock for the return
	expectedGetQuery := `SELECT(.*)FROM assignments a(.*)`
	rows := sqlmock.NewRows([]string{
		"id", "document_id", "document_type",
		"executor_id", "full_name",
		"content", "deadline", "status", "report", "completed_at",
		"created_at", "updated_at",
		"doc_number", "doc_subject",
	}).AddRow(assignID, uuid.New(), "incoming", execID, "Иванов", "Обновленный текст", now, "in_progress", "Отчет", now, now, now, "", "")

	mock.ExpectQuery(expectedGetQuery).WithArgs(assignID).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT(.*)FROM assignment_co_executors(.*)`).WithArgs(assignID).WillReturnRows(sqlmock.NewRows([]string{"id", "login", "full_name"}))

	assign, err := repo.Update(assignID, execID, "Обновленный текст", &now, "in_progress", "Отчет", &now, []string{coExecID.String()})
	require.NoError(t, err)
	require.NotNil(t, assign)
	assert.Equal(t, assignID, assign.ID)
	assert.Equal(t, "Обновленный текст", assign.Content)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAssignmentRepository_GetList(t *testing.T) {
	// Получение списка поручений с фильтрацией и постраничной навигацией
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAssignmentRepository(&database.DB{DB: db})
	now := time.Now()

	filter := models.AssignmentFilter{
		Page:     1,
		PageSize: 10,
	}

	countQuery := `SELECT COUNT\(\*\) FROM assignments a LEFT JOIN incoming_documents inc ON a.document_id = inc.id(.*)`
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	query := `SELECT
			a.id, a.document_id, a.document_type,
			a.executor_id, u_executor.full_name,
			a.content, a.deadline, a.status, a.report, a.completed_at,
			a.created_at, a.updated_at,
			COALESCE\(inc.incoming_number, out.outgoing_number\) as doc_number,
			COALESCE\(inc.content, out.subject\) as doc_subject
		FROM assignments a(.*)`

	mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{
		"id", "document_id", "document_type", "executor_id", "full_name",
		"content", "deadline", "status", "report", "completed_at",
		"created_at", "updated_at", "doc_number", "doc_subject",
	}).AddRow(uuid.New(), uuid.New(), "incoming", uuid.New(), "Executor", "Content", now, "new", nil, nil, now, now, "doc-1", "subj-1"))

	// Co-executors fetching
	mock.ExpectQuery(`SELECT(.*)FROM assignment_co_executors(.*)`).
		WillReturnRows(sqlmock.NewRows([]string{"assignment_id", "user_id", "login", "full_name"}))

	res, err := repo.GetList(filter)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.TotalCount)
	assert.Len(t, res.Items, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

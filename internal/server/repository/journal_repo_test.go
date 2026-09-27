package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJournalRepository_Create(t *testing.T) {
	// Создание новой записи в журнале
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewJournalRepository(database.Wrap(db))
	ctx := context.Background()

	req := models.CreateJournalEntryRequest{
		DocumentID: uuid.New(),
		UserID:     uuid.New(),
		Action:     "TEST_ACTION",
		Details:    "Тестовое описание действия",
	}

	query := `INSERT INTO document_journal .*outbox_deduplication_key.*RETURNING id`

	newID := uuid.New()
	mock.ExpectQuery(query).
		WithArgs(req.DocumentID, req.UserID, req.Action, req.Details, "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(newID))

	id, err := repo.Create(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, newID, id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJournalRepository_CreateFromOutboxTreatsDuplicateAsDelivered(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewJournalRepository(database.Wrap(db))
	req := models.CreateJournalEntryRequest{DocumentID: uuid.New(), UserID: uuid.New(), Action: "TEST", Details: "retry"}
	mock.ExpectExec(`INSERT INTO document_journal`).
		WithArgs(req.DocumentID, req.UserID, req.Action, req.Details, "journal:retry:1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, repo.CreateFromOutbox(context.Background(), req, "journal:retry:1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJournalRepository_GetByDocumentID(t *testing.T) {
	// Получение списка записей журнала для конкретного документа
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewJournalRepository(database.Wrap(db))
	ctx := context.Background()
	docID := uuid.New()
	now := time.Now()

	query := `SELECT j.id, u.full_name, j.action, j.details, j.created_at
		FROM document_journal j
		JOIN users u ON j.user_id = u.id
		WHERE j.document_id = \$1
		ORDER BY j.created_at DESC`

	rows := sqlmock.NewRows([]string{
		"id", "user_name",
		"action", "details", "created_at",
	}).AddRow(
		uuid.New(), "Иванов Иван Иванович",
		"TEST_ACTION", "Тестовое действие", now,
	)

	mock.ExpectQuery(query).WithArgs(docID).WillReturnRows(rows)

	entries, err := repo.GetByDocumentID(ctx, docID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Иванов Иван Иванович", entries[0].UserName)
	assert.Equal(t, "TEST_ACTION", entries[0].Action)
	assert.Equal(t, "Тестовое действие", entries[0].Details)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJournalRepository_GetByDocumentID_Empty(t *testing.T) {
	// Получение пустого списка записей журнала должно возвращать пустой массив, а не nil
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewJournalRepository(database.Wrap(db))
	ctx := context.Background()
	docID := uuid.New()
	query := `SELECT j.id, u.full_name, j.action, j.details, j.created_at
		FROM document_journal j
		JOIN users u ON j.user_id = u.id
		WHERE j.document_id = \$1
		ORDER BY j.created_at DESC`

	// Возвращаем пустой результат
	rows := sqlmock.NewRows([]string{
		"id", "user_name",
		"action", "details", "created_at",
	})

	mock.ExpectQuery(query).WithArgs(docID).WillReturnRows(rows)

	entries, err := repo.GetByDocumentID(ctx, docID)
	require.NoError(t, err)
	require.NotNil(t, entries) // Должен быть пустой массив, а не nil
	require.Len(t, entries, 0)
	require.NoError(t, mock.ExpectationsWereMet())
}

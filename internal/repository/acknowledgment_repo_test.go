package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcknowledgmentRepository_Create(t *testing.T) {
	// Создание листа ознакомления и привязка пользователей
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})

	now := time.Now()
	ack := &models.Acknowledgment{
		ID:         uuid.New(),
		DocumentID: uuid.New(),
		CreatorID:  uuid.New(),
		Content:    "Тест",
		CreatedAt:  now,
		Users: []models.AcknowledgmentUser{
			{ID: uuid.New(), UserID: uuid.New(), CreatedAt: now},
		},
	}

	mock.ExpectBegin()

	mock.ExpectExec(`INSERT INTO acknowledgments`).WithArgs(
		ack.ID, ack.DocumentID, ack.CreatorID, ack.Content, ack.CreatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO acknowledgment_users`).WithArgs(
		ack.Users[0].ID, ack.ID, ack.Users[0].UserID, ack.Users[0].CreatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err = repo.Create(ack)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAcknowledgmentRepository(&database.DB{DB: db})
		ackID := uuid.New()
		docID := uuid.New()
		creatorID := uuid.New()
		now := time.Now()
		completedAt := now.Add(time.Hour)

		query := `SELECT a\.id, a\.document_id, d\.kind, a\.creator_id, a\.content, a\.created_at, a\.completed_at FROM acknowledgments a JOIN documents d ON d\.id = a\.document_id WHERE a\.id = \$1`
		rows := sqlmock.NewRows([]string{
			"id", "document_id", "kind", "creator_id", "content", "created_at", "completed_at",
		}).AddRow(ackID, docID, string(models.DocumentKindIncomingLetter), creatorID, "Ознакомиться", now, completedAt)

		mock.ExpectQuery(query).WithArgs(ackID).WillReturnRows(rows)

		ack, err := repo.GetByID(ackID)
		require.NoError(t, err)
		require.NotNil(t, ack)
		assert.Equal(t, ackID, ack.ID)
		assert.Equal(t, docID, ack.DocumentID)
		assert.Equal(t, string(models.DocumentKindIncomingLetter), ack.DocumentKind)
		assert.Equal(t, creatorID, ack.CreatorID)
		assert.Equal(t, "Ознакомиться", ack.Content)
		assert.Equal(t, now, ack.CreatedAt)
		require.NotNil(t, ack.CompletedAt)
		assert.Equal(t, completedAt, *ack.CompletedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAcknowledgmentRepository(&database.DB{DB: db})
		ackID := uuid.New()

		query := `SELECT a\.id, a\.document_id, d\.kind, a\.creator_id, a\.content, a\.created_at, a\.completed_at FROM acknowledgments a JOIN documents d ON d\.id = a\.document_id WHERE a\.id = \$1`
		mock.ExpectQuery(query).WithArgs(ackID).WillReturnError(sqlmock.ErrCancelled)

		ack, err := repo.GetByID(ackID)
		require.Error(t, err)
		assert.Nil(t, ack)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAcknowledgmentRepository_GetByDocumentID(t *testing.T) {
	// Получение списка листов ознакомления по ID документа
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	docID := uuid.New()
	ackID := uuid.New()
	now := time.Now()

	expectedQuery := `SELECT(.*)FROM acknowledgments a(.*)JOIN documents d ON d.id = a.document_id(.*)WHERE a.document_id = \$1(.*)`

	rows := sqlmock.NewRows([]string{
		"id", "document_id", "kind", "creator_id", "content", "created_at", "completed_at",
		"creator_name", "doc_number",
	}).AddRow(ackID, docID, "incoming", uuid.New(), "Ознакомиться", now, nil, "Создатель", "ВХ-1")

	mock.ExpectQuery(expectedQuery).WithArgs(docID).WillReturnRows(rows)

	usersQuery := `SELECT 
			au.id, au.acknowledgment_id, au.user_id, au.viewed_at, au.confirmed_at, au.created_at,
			u.full_name as user_name
		FROM acknowledgment_users au
		JOIN users u ON au.user_id = u.id
		WHERE au.acknowledgment_id = $1`

	usersRows := sqlmock.NewRows([]string{
		"id", "acknowledgment_id", "user_id", "viewed_at", "confirmed_at", "created_at", "user_name",
	}).AddRow(uuid.New(), ackID, uuid.New(), nil, nil, now, "Читатель")

	mock.ExpectQuery(regexp.QuoteMeta(usersQuery)).WithArgs(ackID).WillReturnRows(usersRows)

	acks, err := repo.GetByDocumentID(docID)
	require.NoError(t, err)
	require.Len(t, acks, 1)
	assert.Equal(t, ackID, acks[0].ID)
	assert.Len(t, acks[0].Users, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_MarkViewed(t *testing.T) {
	// Отметка о прочтении листа ознакомления конкретным пользователем
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	ackID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec(`UPDATE acknowledgment_users SET viewed_at = \$1 WHERE acknowledgment_id = \$2 AND user_id = \$3 AND viewed_at IS NULL`).
		WithArgs(sqlmock.AnyArg(), ackID, userID).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.MarkViewed(ackID, userID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_MarkViewedReturnsForbiddenWhenUserRowMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	ackID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec(`UPDATE acknowledgment_users SET viewed_at = \$1 WHERE acknowledgment_id = \$2 AND user_id = \$3 AND viewed_at IS NULL`).
		WithArgs(sqlmock.AnyArg(), ackID, userID).WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.MarkViewed(ackID, userID)
	require.ErrorIs(t, err, models.ErrForbidden)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_Delete(t *testing.T) {
	// Удаление листа ознакомления по его ID
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	ackID := uuid.New()

	mock.ExpectExec(`DELETE FROM acknowledgments WHERE id = \$1`).WithArgs(ackID).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(ackID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetUsersByAcknowledgmentID(t *testing.T) {
	// Получение списка пользователей, привязанных к листу ознакомления
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	ackID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	query := `SELECT 
			au.id, au.acknowledgment_id, au.user_id, au.viewed_at, au.confirmed_at, au.created_at,
			u.full_name as user_name
		FROM acknowledgment_users au
		JOIN users u ON au.user_id = u.id
		WHERE au.acknowledgment_id = \$1`

	rows := sqlmock.NewRows([]string{
		"id", "acknowledgment_id", "user_id", "viewed_at", "confirmed_at", "created_at", "user_name",
	}).AddRow(uuid.New(), ackID, userID, now, nil, now, "Читатель")

	mock.ExpectQuery(query).WithArgs(ackID).WillReturnRows(rows)

	users, err := repo.GetUsersByAcknowledgmentID(ackID)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, userID, users[0].UserID)
	assert.Equal(t, "Читатель", users[0].UserName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetPendingForUser(t *testing.T) {
	// Получение списка ожидающих подтверждения листов ознакомления для пользователя
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	userID := uuid.New()
	now := time.Now()

	query := `SELECT(.*)FROM acknowledgment_users au(.*)JOIN documents d ON d.id = a.document_id(.*)WHERE au.user_id = \$1 AND au.confirmed_at IS NULL ORDER BY a.created_at DESC`

	rows := sqlmock.NewRows([]string{
		"id", "document_id", "kind", "creator_id", "content", "created_at", "completed_at",
		"creator_name", "doc_number",
	}).AddRow(uuid.New(), uuid.New(), "incoming", uuid.New(), "Ознакомиться", now, nil, "Создатель", "ВХ-1")

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)

	usersQuery := `SELECT 
			au.id, au.acknowledgment_id, au.user_id, au.viewed_at, au.confirmed_at, au.created_at,
			u.full_name as user_name
		FROM acknowledgment_users au
		JOIN users u ON au.user_id = u.id
		WHERE au.acknowledgment_id = \$1`

	mock.ExpectQuery(usersQuery).WithArgs(sqlmock.AnyArg()).WillReturnRows(
		sqlmock.NewRows([]string{"id", "acknowledgment_id", "user_id", "viewed_at", "confirmed_at", "created_at", "user_name"}),
	)

	acks, err := repo.GetPendingForUser(userID)
	require.NoError(t, err)
	require.Len(t, acks, 1)
	assert.Equal(t, "Ознакомиться", acks[0].Content)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_MarkConfirmed(t *testing.T) {
	// Подтверждение факта ознакомления пользователем
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	ackID := uuid.New()
	userID := uuid.New()

	mock.ExpectBegin()

	mock.ExpectExec(`UPDATE acknowledgment_users SET confirmed_at = \$1, viewed_at = COALESCE\(viewed_at, \$1\) WHERE acknowledgment_id = \$2 AND user_id = \$3`).
		WithArgs(sqlmock.AnyArg(), ackID, userID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM acknowledgment_users WHERE acknowledgment_id = \$1 AND confirmed_at IS NULL`).
		WithArgs(ackID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`UPDATE acknowledgments SET completed_at = \$1 WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), ackID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err = repo.MarkConfirmed(ackID, userID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_MarkConfirmedReturnsForbiddenWhenUserRowMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	ackID := uuid.New()
	userID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE acknowledgment_users SET confirmed_at = \$1, viewed_at = COALESCE\(viewed_at, \$1\) WHERE acknowledgment_id = \$2 AND user_id = \$3`).
		WithArgs(sqlmock.AnyArg(), ackID, userID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = repo.MarkConfirmed(ackID, userID)
	require.ErrorIs(t, err, models.ErrForbidden)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetAllActive(t *testing.T) {
	// Получение всех активных (не завершенных) листов ознакомления
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	now := time.Now()

	query := `SELECT(.*)FROM acknowledgments a(.*)JOIN documents d ON d.id = a.document_id(.*)WHERE a.completed_at IS NULL(.*)d.kind = ANY\(\$1\)(.*)`

	rows := sqlmock.NewRows([]string{
		"id", "document_id", "kind", "creator_id", "content", "created_at", "completed_at",
		"creator_name", "doc_number",
	}).AddRow(uuid.New(), uuid.New(), "incoming", uuid.New(), "Ознакомиться", now, nil, "Создатель", "ВХ-1")

	mock.ExpectQuery(query).WithArgs(pq.Array([]string{"incoming_letter"})).WillReturnRows(rows)

	acks, err := repo.GetAllActive(models.AcknowledgmentFilter{AllowedDocumentKinds: []string{"incoming_letter"}})
	require.NoError(t, err)
	require.Len(t, acks, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_DocumentAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	userID := uuid.New()
	docID := uuid.New()

	t.Run("has document access", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(userID, docID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		ok, err := repo.HasDocumentAccess(userID, docID)

		require.NoError(t, err)
		assert.True(t, ok)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty bulk input", func(t *testing.T) {
		result, err := repo.GetAccessibleDocumentIDs(userID, nil)

		require.NoError(t, err)
		assert.Empty(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("bulk accessible ids", func(t *testing.T) {
		allowedID := uuid.New()
		deniedID := uuid.New()

		mock.ExpectQuery(`SELECT DISTINCT a\.document_id\s+FROM acknowledgment_users au`).
			WithArgs(userID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"document_id"}).AddRow(allowedID))

		result, err := repo.GetAccessibleDocumentIDs(userID, []uuid.UUID{allowedID, deniedID})

		require.NoError(t, err)
		assert.Contains(t, result, allowedID)
		assert.NotContains(t, result, deniedID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

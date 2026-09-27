package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcknowledgmentRepository_CreateWithOutbox(t *testing.T) {
	// Создание листа ознакомления и привязка пользователей
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(database.Wrap(db))
	repo.SetOutbox(NewOutboxRepository(repo.db))

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

	err = repo.CreateWithOutbox(ack, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepositoryCreateWithOutboxRollsBackOnEnqueueFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(database.Wrap(db))
	repo.SetOutbox(NewOutboxRepository(repo.db))
	now := time.Now()
	ack := &models.Acknowledgment{
		ID:         uuid.New(),
		DocumentID: uuid.New(),
		CreatorID:  uuid.New(),
		Content:    "Тест",
		CreatedAt:  now,
		Users:      []models.AcknowledgmentUser{{ID: uuid.New(), UserID: uuid.New(), CreatedAt: now}},
	}
	event := models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "ack:" + ack.ID.String(), Payload: `{"action":"ACK_CREATE"}`}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO acknowledgments`).WithArgs(ack.ID, ack.DocumentID, ack.CreatorID, ack.Content, ack.CreatedAt).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO acknowledgment_users`).WithArgs(ack.Users[0].ID, ack.ID, ack.Users[0].UserID, ack.Users[0].CreatedAt).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs(event.EventType, event.DeduplicationKey, event.Payload).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	err = repo.CreateWithOutbox(ack, []models.OutboxEvent{event})
	require.ErrorIs(t, err, assert.AnError)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAcknowledgmentRepository(database.Wrap(db))
		ackID := uuid.New()
		docID := uuid.New()
		creatorID := uuid.New()
		query := `SELECT a\.document_id, d\.kind, a\.creator_id FROM acknowledgments a JOIN documents d ON d\.id = a\.document_id WHERE a\.id = \$1`
		rows := sqlmock.NewRows([]string{
			"document_id", "kind", "creator_id",
		}).AddRow(docID, string(models.DocumentKindIncomingLetter), creatorID)

		mock.ExpectQuery(query).WithArgs(ackID).WillReturnRows(rows)

		ack, err := repo.GetByID(ackID)
		require.NoError(t, err)
		require.NotNil(t, ack)
		assert.Equal(t, ackID, ack.ID)
		assert.Equal(t, docID, ack.DocumentID)
		assert.Equal(t, string(models.DocumentKindIncomingLetter), ack.DocumentKind)
		assert.Equal(t, creatorID, ack.CreatorID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		repo := NewAcknowledgmentRepository(database.Wrap(db))
		ackID := uuid.New()
		mock.ExpectQuery(`SELECT a\.document_id, d\.kind, a\.creator_id`).WithArgs(ackID).
			WillReturnRows(sqlmock.NewRows([]string{"document_id", "kind", "creator_id"}))
		ack, err := repo.GetByID(ackID)
		require.NoError(t, err)
		require.Nil(t, ack)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAcknowledgmentRepository(database.Wrap(db))
		ackID := uuid.New()

		query := `SELECT a\.document_id, d\.kind, a\.creator_id FROM acknowledgments a JOIN documents d ON d\.id = a\.document_id WHERE a\.id = \$1`
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

	repo := NewAcknowledgmentRepository(database.Wrap(db))
	docID := uuid.New()
	ackID := uuid.New()
	now := time.Now()

	expectedQuery := `SELECT(.*)FROM acknowledgments a(.*)JOIN documents d ON d.id = a.document_id(.*)WHERE a.document_id = \$1(.*)`

	rows := sqlmock.NewRows([]string{
		"id", "document_id", "kind", "creator_id", "content", "created_at", "completed_at",
		"creator_name", "doc_number",
	}).AddRow(ackID, docID, "incoming_letter", uuid.New(), "Ознакомиться", now, nil, "Создатель", "ВХ-1").
		AddRow(uuid.New(), docID, "incoming_letter", uuid.New(), "Второе ознакомление", now, nil, "Создатель", "ВХ-1")

	mock.ExpectQuery(expectedQuery).WithArgs(docID).WillReturnRows(rows)

	usersQuery := `SELECT 
			au.acknowledgment_id, au.user_id, au.confirmed_at,
			u.full_name as user_name
		FROM acknowledgment_users au
		JOIN users u ON au.user_id = u.id
		WHERE au.acknowledgment_id = ANY($1)
		ORDER BY au.acknowledgment_id, au.created_at, au.id`

	usersRows := sqlmock.NewRows([]string{
		"acknowledgment_id", "user_id", "confirmed_at", "user_name",
	}).AddRow(ackID, uuid.New(), nil, "Читатель")

	mock.ExpectQuery(regexp.QuoteMeta(usersQuery)).WithArgs(sqlmock.AnyArg()).WillReturnRows(usersRows)

	acks, err := repo.GetByDocumentID(docID)
	require.NoError(t, err)
	require.Len(t, acks, 2)
	assert.Equal(t, ackID, acks[0].ID)
	assert.Len(t, acks[0].Users, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_DeleteWithOutbox(t *testing.T) {
	// Удаление листа ознакомления по его ID
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(database.Wrap(db))
	repo.SetOutbox(NewOutboxRepository(repo.db))
	ackID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM acknowledgments WHERE id = \$1`).WithArgs(ackID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err = repo.DeleteWithOutbox(ackID, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepositoryDeleteWithOutboxRollsBackOnEnqueueFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(database.Wrap(db))
	repo.SetOutbox(NewOutboxRepository(repo.db))
	ackID := uuid.New()
	event := models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "ack:" + ackID.String() + ":delete", Payload: `{"action":"ACK_DELETE"}`}

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM acknowledgments WHERE id = \$1`).WithArgs(ackID).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs(event.EventType, event.DeduplicationKey, event.Payload).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	err = repo.DeleteWithOutbox(ackID, []models.OutboxEvent{event})
	require.ErrorIs(t, err, assert.AnError)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetUsersByAcknowledgmentIDs(t *testing.T) {
	// Получение списка пользователей, привязанных к листу ознакомления
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(database.Wrap(db))
	ackID := uuid.New()
	secondAckID := uuid.New()
	userID := uuid.New()

	query := `SELECT 
			au.acknowledgment_id, au.user_id, au.confirmed_at,
			u.full_name as user_name
		FROM acknowledgment_users au
		JOIN users u ON au.user_id = u.id
		WHERE au.acknowledgment_id = ANY\(\$1\)
		ORDER BY au.acknowledgment_id, au.created_at, au.id`

	rows := sqlmock.NewRows([]string{
		"acknowledgment_id", "user_id", "confirmed_at", "user_name",
	}).AddRow(ackID, userID, nil, "Читатель").AddRow(secondAckID, uuid.New(), nil, "Второй")

	mock.ExpectQuery(query).WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)

	users, err := repo.GetUsersByAcknowledgmentIDs([]uuid.UUID{ackID, secondAckID})
	require.NoError(t, err)
	require.Len(t, users[ackID], 1)
	assert.Equal(t, userID, users[ackID][0].UserID)
	assert.Equal(t, "Читатель", users[ackID][0].UserName)
	require.Len(t, users[secondAckID], 1)
	assert.Equal(t, "Второй", users[secondAckID][0].UserName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetPendingForUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(database.Wrap(db))
	userID := uuid.New()
	principalID := uuid.New()
	ackID := uuid.New()
	docID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT(.*)au.user_id(.*)WHERE au.user_id = ANY\(\$1\) AND au.confirmed_at IS NULL`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "id", "document_id", "kind", "creator_id", "content", "created_at", "completed_at", "creator_name", "doc_number",
		}).AddRow(principalID, ackID, docID, "incoming_letter", uuid.New(), "Ознакомиться", now, nil, "Создатель", "ВХ-1"))
	items, err := repo.GetPendingForUsers([]uuid.UUID{userID, principalID}, uuid.Nil)
	require.NoError(t, err)
	require.Len(t, items[principalID], 1)
	assert.Equal(t, ackID, items[principalID][0].ID)
	assert.Nil(t, items[principalID][0].Users)
	assert.Empty(t, items[userID])
	mock.ExpectQuery(`SELECT(.*)au.user_id(.*)WHERE au.user_id = ANY\(\$1\) AND au.confirmed_at IS NULL(.*)AND a.document_id = \$2`).
		WithArgs(sqlmock.AnyArg(), docID).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "id", "document_id", "kind", "creator_id", "content", "created_at", "completed_at", "creator_name", "doc_number",
		}).AddRow(principalID, ackID, docID, "incoming_letter", uuid.New(), "Ознакомиться", now, nil, "Создатель", "ВХ-1"))
	filtered, err := repo.GetPendingForUsers([]uuid.UUID{userID, principalID}, docID)
	require.NoError(t, err)
	require.Len(t, filtered[principalID], 1)
	assert.Equal(t, docID, filtered[principalID][0].DocumentID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetPendingRecipientIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewAcknowledgmentRepository(database.Wrap(db))
	ackID, userID, otherID := uuid.New(), uuid.New(), uuid.New()
	mock.ExpectQuery(`SELECT user_id FROM acknowledgment_users WHERE acknowledgment_id = \$1 AND user_id = ANY\(\$2\) AND confirmed_at IS NULL`).
		WithArgs(ackID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))
	found, err := repo.GetPendingRecipientIDs(ackID, []uuid.UUID{userID, otherID})
	require.NoError(t, err)
	require.Equal(t, map[uuid.UUID]struct{}{userID: {}}, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetAllActive(t *testing.T) {
	// Получение всех активных (не завершенных) листов ознакомления
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(database.Wrap(db))
	now := time.Now()

	query := `SELECT(.*)FROM acknowledgments a(.*)JOIN documents d ON d.id = a.document_id(.*)WHERE a.completed_at IS NULL(.*)d.kind = ANY\(\$1\)(.*)`

	rows := sqlmock.NewRows([]string{
		"id", "document_id", "kind", "creator_id", "content", "created_at", "completed_at",
		"creator_name", "doc_number",
	}).AddRow(uuid.New(), uuid.New(), "incoming_letter", uuid.New(), "Ознакомиться", now, nil, "Создатель", "ВХ-1")

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

	repo := NewAcknowledgmentRepository(database.Wrap(db))
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

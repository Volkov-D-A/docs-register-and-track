package repository

import (
	"regexp"
	"testing"
	"time"

	"docflow/internal/database"
	"docflow/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcknowledgmentRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})

	now := time.Now()
	ack := &models.Acknowledgment{
		ID:           uuid.New(),
		DocumentID:   uuid.New(),
		DocumentType: "incoming",
		CreatorID:    uuid.New(),
		Content:      "Тест",
		CreatedAt:    now,
		Users: []models.AcknowledgmentUser{
			{ID: uuid.New(), UserID: uuid.New(), CreatedAt: now},
		},
	}

	mock.ExpectBegin()

	mock.ExpectExec(`INSERT INTO acknowledgments`).WithArgs(
		ack.ID, ack.DocumentID, ack.DocumentType, ack.CreatorID, ack.Content, ack.CreatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO acknowledgment_users`).WithArgs(
		ack.Users[0].ID, ack.ID, ack.Users[0].UserID, ack.Users[0].CreatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err = repo.Create(ack)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcknowledgmentRepository_GetByDocumentID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAcknowledgmentRepository(&database.DB{DB: db})
	docID := uuid.New()
	ackID := uuid.New()
	now := time.Now()

	expectedQuery := `SELECT 
			a.id, a.document_id, a.document_type, a.creator_id, a.content, a.created_at, a.completed_at,
			u.full_name as creator_name,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number
		FROM acknowledgments a
		JOIN users u ON a.creator_id = u.id
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
		WHERE a.document_id = $1
		ORDER BY a.created_at DESC`

	rows := sqlmock.NewRows([]string{
		"id", "document_id", "document_type", "creator_id", "content", "created_at", "completed_at",
		"creator_name", "doc_number",
	}).AddRow(ackID, docID, "incoming", uuid.New(), "Ознакомиться", now, nil, "Создатель", "ВХ-1")

	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

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

func TestAcknowledgmentRepository_Delete(t *testing.T) {
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

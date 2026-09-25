package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

func TestUserEventRepository_CreateFromOutboxDoesNotReloadEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	request := models.CreateUserEventRequest{
		RecipientUserID: uuid.New(), DocumentID: uuid.New(),
		DocumentKind: "incoming_letter", EntityType: models.UserEventEntityAssignment,
		EventType: models.UserEventAssignmentCreated, Title: "Новое поручение", Message: "Назначено поручение",
	}
	mock.ExpectExec(`INSERT INTO user_events`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = NewUserEventRepository(&database.DB{DB: db}).CreateFromOutbox(request, "assignment:created:user-event")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserEventRepository_GetList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserEventRepository(&database.DB{DB: db})
	userID := uuid.New()
	eventID := uuid.New()
	documentID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM user_events e WHERE e.recipient_user_id = $1 AND e.read_at IS NULL")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	listQuery := `SELECT
			e.id, e.recipient_user_id,
			e.document_id, e.document_kind, e.document_number,
			e.entity_type, e.event_type,
			e.title, e.message,
			e.created_at, e.read_at
		FROM user_events e
		WHERE e.recipient_user_id = $1 AND e.read_at IS NULL
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3`
	mock.ExpectQuery(regexp.QuoteMeta(listQuery)).
		WithArgs(userID, 10, 10).
		WillReturnRows(sqlmock.NewRows(userEventColumns()).AddRow(
			eventID,
			userID,
			documentID,
			"incoming_letter",
			"ВХ-1",
			models.UserEventEntityAssignment,
			models.UserEventAssignmentCompleted,
			"Поручение ожидает приемки",
			"Исполнитель отправил поручение",
			now,
			nil,
		))

	result, err := repo.GetList(userID, models.UserEventFilter{UnreadOnly: true, Page: 2, PageSize: 10})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, 2, result.Page)
	require.Len(t, result.Items, 1)
	assert.Equal(t, models.UserEventAssignmentCompleted, result.Items[0].EventType)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserEventRepository_MarkRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserEventRepository(&database.DB{DB: db})
	eventID := uuid.New()
	userID := uuid.New()
	readAt := time.Now()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_events SET read_at = COALESCE(read_at, $3) WHERE id = $1 AND recipient_user_id = $2")).
		WithArgs(eventID, userID, readAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.MarkRead(eventID, userID, readAt)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserEventRepository_MarkDocumentRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserEventRepository(&database.DB{DB: db})
	documentID := uuid.New()
	userID := uuid.New()
	readAt := time.Now()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_events SET read_at = COALESCE(read_at, $3) WHERE document_id = $1 AND recipient_user_id = $2 AND read_at IS NULL")).
		WithArgs(documentID, userID, readAt).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err = repo.MarkDocumentRead(documentID, userID, readAt)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func userEventColumns() []string {
	return []string{
		"id",
		"recipient_user_id",
		"document_id",
		"document_kind",
		"document_number",
		"entity_type",
		"event_type",
		"title",
		"message",
		"created_at",
		"read_at",
	}
}

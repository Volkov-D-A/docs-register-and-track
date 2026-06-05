package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestUserEventRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserEventRepository(&database.DB{DB: db})
	eventID := uuid.New()
	recipientID := uuid.New()
	actorID := uuid.New()
	documentID := uuid.New()
	entityID := uuid.New()
	now := time.Now()

	insertQuery := `INSERT INTO user_events (
			recipient_user_id, actor_user_id, document_id, document_kind,
			document_number, entity_type, entity_id, event_type,
			title, message, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
		RETURNING id`
	mock.ExpectQuery(regexp.QuoteMeta(insertQuery)).
		WithArgs(
			recipientID,
			&actorID,
			documentID,
			"incoming_letter",
			"ВХ-1",
			models.UserEventEntityAssignment,
			entityID,
			models.UserEventAssignmentCreated,
			"Новое поручение",
			"Вам назначено поручение",
			"{}",
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(eventID))

	expectGetUserEventByID(mock, eventID, userEventRow{
		ID:              eventID,
		RecipientUserID: recipientID,
		ActorUserID:     actorID,
		ActorUserName:   "Автор",
		DocumentID:      documentID,
		DocumentKind:    "incoming_letter",
		DocumentNumber:  "ВХ-1",
		EntityType:      models.UserEventEntityAssignment,
		EntityID:        entityID,
		EventType:       models.UserEventAssignmentCreated,
		Title:           "Новое поручение",
		Message:         "Вам назначено поручение",
		Metadata:        "{}",
		CreatedAt:       now,
	})

	event, err := repo.Create(models.CreateUserEventRequest{
		RecipientUserID: recipientID,
		ActorUserID:     &actorID,
		DocumentID:      documentID,
		DocumentKind:    "incoming_letter",
		DocumentNumber:  "ВХ-1",
		EntityType:      models.UserEventEntityAssignment,
		EntityID:        entityID,
		EventType:       models.UserEventAssignmentCreated,
		Title:           "Новое поручение",
		Message:         "Вам назначено поручение",
	})

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, eventID, event.ID)
	assert.Equal(t, "Автор", event.ActorUserName)
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
	entityID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM user_events e WHERE e.recipient_user_id = $1 AND e.read_at IS NULL")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	listQuery := `SELECT
			e.id, e.recipient_user_id, e.actor_user_id, actor.full_name,
			e.document_id, e.document_kind, e.document_number,
			e.entity_type, e.entity_id, e.event_type,
			e.title, e.message, e.metadata::text,
			e.created_at, e.read_at
		FROM user_events e
		LEFT JOIN users actor ON actor.id = e.actor_user_id
		WHERE e.recipient_user_id = $1 AND e.read_at IS NULL
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3`
	mock.ExpectQuery(regexp.QuoteMeta(listQuery)).
		WithArgs(userID, 10, 10).
		WillReturnRows(sqlmock.NewRows(userEventColumns()).AddRow(
			eventID,
			userID,
			nil,
			nil,
			documentID,
			"incoming_letter",
			"ВХ-1",
			models.UserEventEntityAssignment,
			entityID,
			models.UserEventAssignmentCompleted,
			"Поручение ожидает приемки",
			"Исполнитель отправил поручение",
			`{"status":"completed"}`,
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

type userEventRow struct {
	ID              uuid.UUID
	RecipientUserID uuid.UUID
	ActorUserID     uuid.UUID
	ActorUserName   string
	DocumentID      uuid.UUID
	DocumentKind    string
	DocumentNumber  string
	EntityType      string
	EntityID        uuid.UUID
	EventType       string
	Title           string
	Message         string
	Metadata        string
	CreatedAt       time.Time
	ReadAt          *time.Time
}

func userEventColumns() []string {
	return []string{
		"id",
		"recipient_user_id",
		"actor_user_id",
		"full_name",
		"document_id",
		"document_kind",
		"document_number",
		"entity_type",
		"entity_id",
		"event_type",
		"title",
		"message",
		"metadata",
		"created_at",
		"read_at",
	}
}

func expectGetUserEventByID(mock sqlmock.Sqlmock, id uuid.UUID, row userEventRow) {
	query := `SELECT
			e.id, e.recipient_user_id, e.actor_user_id, actor.full_name,
			e.document_id, e.document_kind, e.document_number,
			e.entity_type, e.entity_id, e.event_type,
			e.title, e.message, e.metadata::text,
			e.created_at, e.read_at
		FROM user_events e
		LEFT JOIN users actor ON actor.id = e.actor_user_id
		WHERE e.id = $1`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(userEventColumns()).AddRow(
			row.ID,
			row.RecipientUserID,
			row.ActorUserID.String(),
			row.ActorUserName,
			row.DocumentID,
			row.DocumentKind,
			row.DocumentNumber,
			row.EntityType,
			row.EntityID,
			row.EventType,
			row.Title,
			row.Message,
			row.Metadata,
			row.CreatedAt,
			row.ReadAt,
		))
}

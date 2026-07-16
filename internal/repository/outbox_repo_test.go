package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestOutboxRepositoryEnqueueTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewOutboxRepository(&database.DB{DB: db})
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs("user_event", "ack:1:confirmed:2", `{"version":1}`).WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.EnqueueTx(tx, models.OutboxEvent{EventType: "user_event", DeduplicationKey: "ack:1:confirmed:2", Payload: `{"version":1}`}))
	mock.ExpectCommit()
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepositoryClaimPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewOutboxRepository(&database.DB{DB: db})
	id := uuid.New()
	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "available_at", "processing_started_at", "processed_at", "failed_at", "attempts", "last_error", "created_at"}).
			AddRow(id, models.OutboxEventUserEvent, "key", `{"request":{}}`, now, now, nil, nil, 1, nil, now))
	mock.ExpectCommit()

	events, err := repo.ClaimPending(10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, id, events[0].ID)
	require.Equal(t, 1, events[0].Attempts)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepositoryMarkFailedAndRequeue(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewOutboxRepository(&database.DB{DB: db})
	id := uuid.New()
	mock.ExpectExec(`UPDATE event_outbox.*failed_at = CASE`).
		WithArgs(id, 10, 10, 3600.0, "unavailable").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.MarkFailed(id, 10, time.Hour, 10, "unavailable"))
	mock.ExpectExec(`UPDATE event_outbox.*failed_at = NULL`).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Requeue(id))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepositoryStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewOutboxRepository(&database.DB{DB: db})
	mock.ExpectQuery(`SELECT\s+COUNT\(\*\) FILTER`).
		WillReturnRows(sqlmock.NewRows([]string{"pending", "processing", "failed", "processed"}).AddRow(2, 1, 3, 4))
	stats, err := repo.Stats()
	require.NoError(t, err)
	require.Equal(t, models.OutboxStats{Pending: 2, Processing: 1, Failed: 3, Processed: 4}, stats)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepositoryRequiredAuditStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewOutboxRepository(&database.DB{DB: db})
	mock.ExpectQuery(`FROM event_outbox\s+WHERE event_type IN`).
		WithArgs(models.OutboxEventJournal, models.OutboxEventAudit).
		WillReturnRows(sqlmock.NewRows([]string{"pending", "processing", "failed"}).AddRow(2, 1, 3))
	stats, err := repo.RequiredAuditStats()
	require.NoError(t, err)
	require.Equal(t, models.RequiredAuditStats{Pending: 2, Processing: 1, Failed: 3}, stats)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepositoryGetFailed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewOutboxRepository(&database.DB{DB: db})
	id, now := uuid.New(), time.Now()
	mock.ExpectQuery(`FROM event_outbox WHERE failed_at IS NOT NULL`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "attempts", "last_error", "created_at", "failed_at"}).
			AddRow(id, models.OutboxEventAudit, "audit:1", 10, "storage unavailable", now, now))
	events, err := repo.GetFailed(0)
	require.NoError(t, err)
	require.Equal(t, []models.FailedOutboxEvent{{ID: id, EventType: models.OutboxEventAudit, DeduplicationKey: "audit:1", Attempts: 10, LastError: "storage unavailable", CreatedAt: now, FailedAt: now}}, events)
	require.NoError(t, mock.ExpectationsWereMet())
}

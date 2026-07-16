package outbox

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
)

type fileDeleterStub struct {
	path string
	err  error
}

func (s *fileDeleterStub) DeleteFile(_ context.Context, path string) error {
	s.path = path
	return s.err
}

func TestWorkerProcessOnceMarksAlreadyDeliveredUserEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := &database.DB{DB: db}
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped), nil, nil)
	id, now := uuid.New(), time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "available_at", "processing_started_at", "processed_at", "failed_at", "attempts", "last_error", "created_at"}).
			AddRow(id, models.OutboxEventUserEvent, "event-key", `{"request":{}}`, now, now, nil, nil, 1, nil, now))
	mock.ExpectCommit()
	mock.ExpectQuery(`INSERT INTO user_events`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`UPDATE event_outbox SET processed_at = CURRENT_TIMESTAMP`).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, worker.ProcessOnce())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerProcessOnceSchedulesRetryForUnsupportedEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := &database.DB{DB: db}
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped), nil, nil)
	id, now := uuid.New(), time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "available_at", "processing_started_at", "processed_at", "failed_at", "attempts", "last_error", "created_at"}).
			AddRow(id, "unknown", "event-key", `{}`, now, now, nil, nil, 1, nil, now))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE event_outbox.*failed_at = CASE`).WithArgs(id, 1, 10, 1.0, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, worker.ProcessOnce())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerProcessOnceDeliversAdministrativeAudit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := &database.DB{DB: db}
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped), nil, nil)
	id, userID, now := uuid.New(), uuid.New(), time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "available_at", "processing_started_at", "processed_at", "failed_at", "attempts", "last_error", "created_at"}).
			AddRow(id, models.OutboxEventAudit, "audit-key", `{"UserID":"`+userID.String()+`","UserName":"Admin","Action":"SETTINGS_UPDATE","Details":"changed"}`, now, now, nil, nil, 1, nil, now))
	mock.ExpectCommit()
	mock.ExpectQuery(`INSERT INTO admin_audit_log`).WithArgs(userID, "Admin", "SETTINGS_UPDATE", "changed").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectExec(`UPDATE event_outbox SET processed_at = CURRENT_TIMESTAMP`).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, worker.ProcessOnce())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerProcessOnceDeletesAttachmentObjectAndMarkedRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := &database.DB{DB: db}
	storage := &fileDeleterStub{}
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped), repository.NewAttachmentRepository(wrapped), storage)
	eventID, attachmentID, now := uuid.New(), uuid.New(), time.Now()
	payload := `{"attachmentId":"` + attachmentID.String() + `","storagePath":"attachments/report.pdf"}`
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "available_at", "processing_started_at", "processed_at", "failed_at", "attempts", "last_error", "created_at"}).
			AddRow(eventID, models.OutboxEventFileDelete, "attachment-key", payload, now, now, nil, nil, 1, nil, now))
	mock.ExpectCommit()
	mock.ExpectExec(`DELETE FROM attachments WHERE id = \$1 AND deletion_requested_at IS NOT NULL`).WithArgs(attachmentID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE event_outbox SET processed_at = CURRENT_TIMESTAMP`).WithArgs(eventID).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, worker.ProcessOnce())
	require.Equal(t, "attachments/report.pdf", storage.path)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerProcessOnceRetriesAttachmentDeletionAfterStorageFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := &database.DB{DB: db}
	storage := &fileDeleterStub{err: sql.ErrConnDone}
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped), repository.NewAttachmentRepository(wrapped), storage)
	eventID, attachmentID, now := uuid.New(), uuid.New(), time.Now()
	payload := `{"attachmentId":"` + attachmentID.String() + `","storagePath":"attachments/retry.pdf"}`
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "available_at", "processing_started_at", "processed_at", "failed_at", "attempts", "last_error", "created_at"}).
			AddRow(eventID, models.OutboxEventFileDelete, "attachment-key", payload, now, now, nil, nil, 1, nil, now))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE event_outbox.*failed_at = CASE`).WithArgs(eventID, 1, 10, 1.0, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, worker.ProcessOnce())
	require.Equal(t, "attachments/retry.pdf", storage.path)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerRunReleasesStaleClaimsAndStopsOnContextCancellation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := &database.DB{DB: db}
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped), nil, nil)
	mock.ExpectExec(`UPDATE event_outbox SET processing_started_at = NULL`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "available_at", "processing_started_at", "processed_at", "failed_at", "attempts", "last_error", "created_at"}))
	mock.ExpectCommit()
	mock.ExpectQuery(`FROM event_outbox\s+WHERE event_type IN`).WithArgs(models.OutboxEventJournal, models.OutboxEventAudit).
		WillReturnRows(sqlmock.NewRows([]string{"pending", "processing", "failed"}).AddRow(0, 0, 0))
	mock.ExpectQuery(`SELECT\s+COUNT\(\*\) FILTER`).
		WillReturnRows(sqlmock.NewRows([]string{"pending", "processing", "failed", "processed"}).AddRow(0, 0, 0, 0))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	worker.Run(ctx)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTwoWorkersClaimIndependently(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := &database.DB{DB: db}
	newWorker := func() *Worker {
		return NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped), nil, nil)
	}
	for range 2 {
		mock.ExpectBegin()
		mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
			WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "available_at", "processing_started_at", "processed_at", "failed_at", "attempts", "last_error", "created_at"}))
		mock.ExpectCommit()
	}

	require.NoError(t, newWorker().ProcessOnce())
	require.NoError(t, newWorker().ProcessOnce())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRetryDelayIsBoundedExponential(t *testing.T) {
	require.Equal(t, time.Second, retryDelay(1))
	require.Equal(t, 8*time.Second, retryDelay(4))
	require.Equal(t, maxRetryDelay, retryDelay(100))
}

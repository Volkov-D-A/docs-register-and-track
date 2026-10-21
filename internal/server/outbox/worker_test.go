package outbox

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/liveevents"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
)

func TestWorkerProcessOnceMarksAlreadyDeliveredUserEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := database.Wrap(db)
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped))
	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "attempts"}).
			AddRow(id, models.OutboxEventUserEvent, "event-key", `{"request":{}}`, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`INSERT INTO user_events`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE event_outbox SET processed_at = CURRENT_TIMESTAMP`).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, worker.ProcessOnce())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerProcessOnceSchedulesRetryForUnsupportedEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := database.Wrap(db)
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped))
	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "attempts"}).
			AddRow(id, "unknown", "event-key", `{}`, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE event_outbox.*failed_at = CASE`).WithArgs(id, 1, 10, 1.0, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, worker.ProcessOnce())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerProcessOnceDeliversAdministrativeAudit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := database.Wrap(db)
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped))
	id, userID := uuid.New(), uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "attempts"}).
			AddRow(id, models.OutboxEventAudit, "audit-key", `{"UserID":"`+userID.String()+`","UserName":"Admin","Action":"SETTINGS_UPDATE","Details":"changed"}`, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`INSERT INTO admin_audit_log`).WithArgs(userID, "Admin", "SETTINGS_UPDATE", "changed", "audit-key").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE event_outbox SET processed_at = CURRENT_TIMESTAMP`).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, worker.ProcessOnce())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerRunReleasesStaleClaimsAndStopsOnContextCancellation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := database.Wrap(db)
	worker := NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped))
	mock.ExpectExec(`UPDATE event_outbox SET processing_started_at = NULL`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "attempts"}))
	mock.ExpectCommit()
	mock.ExpectQuery(`FROM event_outbox\s+WHERE event_type IN`).WithArgs(models.OutboxEventJournal, models.OutboxEventAudit).
		WillReturnRows(sqlmock.NewRows([]string{"pending", "processing", "failed"}).AddRow(0, 0, 0))
	mock.ExpectQuery(`FROM event_outbox\s+WHERE processed_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"pending", "processing", "failed"}).AddRow(0, 0, 0))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	worker.Run(ctx)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWorkerCleanupProcessedUsesRetentionAndBoundedBatches(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := database.Wrap(db)
	worker := NewWorker(repository.NewOutboxRepository(wrapped), nil, nil, nil)
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	worker.now = func() time.Time { return now }
	cutoff := now.Add(-worker.options.ProcessedRetention)
	mock.ExpectExec(`WITH expired AS \(.*DELETE FROM event_outbox`).WithArgs(cutoff, cleanupBatchSize).WillReturnResult(sqlmock.NewResult(0, cleanupBatchSize))
	mock.ExpectExec(`WITH expired AS \(.*DELETE FROM event_outbox`).WithArgs(cutoff, cleanupBatchSize).WillReturnResult(sqlmock.NewResult(0, 25))

	worker.cleanupProcessed(context.Background())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTwoWorkersClaimIndependently(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	wrapped := database.Wrap(db)
	newWorker := func() *Worker {
		return NewWorker(repository.NewOutboxRepository(wrapped), repository.NewUserEventRepository(wrapped), repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped))
	}
	for range 2 {
		mock.ExpectBegin()
		mock.ExpectQuery(`WITH due AS \(.*FOR UPDATE SKIP LOCKED.*UPDATE event_outbox`).WithArgs(50).
			WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "deduplication_key", "payload", "attempts"}))
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

func TestWorkerOptionsDefaultsAndValidation(t *testing.T) {
	defaults := (Options{}).WithDefaults()
	require.Equal(t, 50, defaults.BatchSize)
	require.Equal(t, 5*time.Second, defaults.PollingInterval)
	require.NoError(t, defaults.Validate())

	invalid := defaults
	invalid.BatchSize = 1001
	require.Error(t, invalid.Validate())
}

func TestWorkerNotifiesDocumentsOnlyAfterDurableJournalOrAuditDelivery(t *testing.T) {
	for _, eventType := range []string{models.OutboxEventJournal, models.OutboxEventAudit} {
		for _, success := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", eventType, success), func(t *testing.T) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				defer db.Close()
				wrapped := database.Wrap(db)
				worker := NewWorker(repository.NewOutboxRepository(wrapped), nil, repository.NewJournalRepository(wrapped), repository.NewAdminAuditLogRepository(wrapped))
				notifications := 0
				worker.OnDocumentChanged = func(liveevents.Change) { notifications++ }
				worker.OnAccessChanged = func(models.CreateAdminAuditLogRequest) { notifications++ }
				query := "INSERT INTO document_journal"
				if eventType == models.OutboxEventAudit {
					query = "INSERT INTO admin_audit_log"
				}
				expectation := mock.ExpectExec(query)
				if success {
					expectation.WillReturnResult(sqlmock.NewResult(0, 1))
				} else {
					expectation.WillReturnError(errors.New("write failed"))
				}
				err = worker.process(context.Background(), models.OutboxEvent{EventType: eventType, DeduplicationKey: "key", Payload: `{}`})
				if success {
					require.NoError(t, err)
					require.Equal(t, 1, notifications)
				} else {
					require.Error(t, err)
					require.Zero(t, notifications)
				}
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	}
}

func TestJournalChangesIdentifyOnlyAffectedResources(t *testing.T) {
	documentID, readerID := uuid.New(), uuid.New()
	for _, tc := range []struct {
		action, resource string
		visibility       bool
	}{
		{"CREATE", "document", false}, {"UPDATE", "document", false},
		{"DEADLINE_CONTROL_CREATE", "controls", true}, {"DEADLINE_CONTROL_UPDATE", "controls", true}, {"DEADLINE_CONTROL_CANCEL", "controls", true}, {"DEADLINE_CONTROL_COMPLETE", "controls", false}, {"LINK_DELETE", "links", true},
	} {
		t.Run(tc.action, func(t *testing.T) {
			change := journalChange(models.CreateJournalEntryRequest{DocumentID: documentID, Action: tc.action, PreviousReaderIDs: []uuid.UUID{readerID}})
			require.Equal(t, liveevents.Change{DocumentID: documentID.String(), Resource: tc.resource, VisibilityChanged: tc.visibility, PreviousReaders: []uuid.UUID{readerID}}, change)
		})
	}
}

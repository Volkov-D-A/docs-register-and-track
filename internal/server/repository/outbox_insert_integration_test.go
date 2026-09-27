package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestJournalAndAuditOutboxInsertsAreIdempotentIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	userID, documentID := seedIntegrationDocument(t, sqlDB)
	journal := NewJournalRepository(db)
	audit := NewAdminAuditLogRepository(db)
	journalRequest := models.CreateJournalEntryRequest{DocumentID: documentID, UserID: userID, Action: "TEST", Details: "delivered"}
	auditRequest := models.CreateAdminAuditLogRequest{UserID: userID, UserName: "Tester", Action: "TEST", Details: "delivered"}

	for range 2 {
		require.NoError(t, journal.CreateFromOutbox(context.Background(), journalRequest, "journal:integration:retry"))
		require.NoError(t, audit.CreateFromOutbox(auditRequest, "audit:integration:retry"))
	}
	var journalCount, auditCount int
	require.NoError(t, sqlDB.QueryRow(`SELECT COUNT(*) FROM document_journal WHERE outbox_deduplication_key = $1`, "journal:integration:retry").Scan(&journalCount))
	require.NoError(t, sqlDB.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE outbox_deduplication_key = $1`, "audit:integration:retry").Scan(&auditCount))
	require.Equal(t, 1, journalCount)
	require.Equal(t, 1, auditCount)

	badJournal := journalRequest
	badJournal.DocumentID = uuid.New()
	require.Error(t, journal.CreateFromOutbox(context.Background(), badJournal, "journal:integration:invalid"))
	badAudit := auditRequest
	badAudit.UserID = uuid.New()
	require.Error(t, audit.CreateFromOutbox(badAudit, "audit:integration:invalid"))
}

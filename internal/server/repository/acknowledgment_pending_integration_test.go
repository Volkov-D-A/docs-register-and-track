package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestAcknowledgmentPendingByDocumentIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	userID, firstDocumentID := seedIntegrationDocument(t, sqlDB)
	principalID := insertIntegrationUser(t, sqlDB, "ack_principal")
	secondDocumentID := uuid.New()
	execSQL(t, sqlDB, `
		INSERT INTO documents (id, kind, nomenclature_id, idempotency_key, registration_number,
			registration_date, document_type, content, pages_count, created_by)
		SELECT $1, kind, nomenclature_id, $2, 'IT/2', registration_date,
			document_type, 'second document', pages_count, created_by
		FROM documents WHERE id = $3
	`, secondDocumentID, uuid.New(), firstDocumentID)

	firstAckID, delegatedAckID, otherAckID := uuid.New(), uuid.New(), uuid.New()
	for _, item := range []struct {
		ackID, docID, recipientID uuid.UUID
	}{
		{firstAckID, firstDocumentID, userID},
		{delegatedAckID, firstDocumentID, principalID},
		{otherAckID, secondDocumentID, userID},
	} {
		execSQL(t, sqlDB, `INSERT INTO acknowledgments (id, document_id, creator_id, content)
			VALUES ($1, $2, $3, 'read')`, item.ackID, item.docID, userID)
		execSQL(t, sqlDB, `INSERT INTO acknowledgment_users (id, acknowledgment_id, user_id)
			VALUES ($1, $2, $3)`, uuid.New(), item.ackID, item.recipientID)
	}

	repo := NewAcknowledgmentRepository(database.Wrap(sqlDB))
	all, err := repo.GetPendingForUsers([]uuid.UUID{userID, principalID}, uuid.Nil)
	require.NoError(t, err)
	require.Len(t, all[userID], 2)
	require.Len(t, all[principalID], 1)

	filtered, err := repo.GetPendingForUsers([]uuid.UUID{userID, principalID}, firstDocumentID)
	require.NoError(t, err)
	require.Len(t, filtered[userID], 1)
	require.Equal(t, firstAckID, filtered[userID][0].ID)
	require.Len(t, filtered[principalID], 1)
	require.Equal(t, delegatedAckID, filtered[principalID][0].ID)

	other, err := repo.GetPendingForUsers([]uuid.UUID{userID, principalID}, secondDocumentID)
	require.NoError(t, err)
	require.Len(t, other[userID], 1)
	require.Equal(t, otherAckID, other[userID][0].ID)
	require.Empty(t, other[principalID])

	pendingRecipients, err := repo.GetPendingRecipientIDs(delegatedAckID, []uuid.UUID{userID, principalID})
	require.NoError(t, err)
	require.Equal(t, map[uuid.UUID]struct{}{principalID: {}}, pendingRecipients)
	pendingRecipients, err = repo.GetPendingRecipientIDs(firstAckID, []uuid.UUID{principalID})
	require.NoError(t, err)
	require.Empty(t, pendingRecipients)
}

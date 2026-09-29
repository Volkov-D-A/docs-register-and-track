package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestAssignmentAcknowledgmentConcurrentConfirmationIntegration(t *testing.T) {
	db := integrationdb.Open(t)
	creator, document := seedIntegrationDocument(t, db)
	second := insertIntegrationUser(t, db, "ack-second")
	actor := insertIntegrationUser(t, db, "ack-substitute")
	id := uuid.New()
	repo := NewAssignmentRepository(database.Wrap(db))
	repo.SetOutbox(NewOutboxRepository(database.Wrap(db)))
	due := time.Now().UTC().AddDate(0, 0, 3)
	now := time.Now().UTC()
	require.NoError(t, repo.CreateRecipientTaskWithOutbox(&models.Assignment{
		ID: id, DocumentID: document, CreatorID: creator, CreatedAt: now, Deadline: &due,
		Users: []models.AssignmentRecipient{{ID: uuid.New(), UserID: creator, CreatedAt: now}, {ID: uuid.New(), UserID: second, CreatedAt: now}},
	}, nil))
	allowed, err := repo.HasDocumentAccess(second, document)
	require.NoError(t, err)
	require.True(t, allowed)
	readable, err := repo.GetAccessibleDocumentIDs(second, []uuid.UUID{document})
	require.NoError(t, err)
	require.Contains(t, readable, document)
	allowed, err = repo.HasDocumentAccess(actor, document)
	require.NoError(t, err)
	require.False(t, allowed)
	statistics, err := NewStatisticsRepository(database.Wrap(db)).GetAssignmentStatusCounts()
	require.NoError(t, err)
	require.Empty(t, statistics)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, subject := range []uuid.UUID{creator, second} {
		go func(subject uuid.UUID) {
			<-start
			results <- repo.ConfirmRecipientWithEffects(id, subject, models.AssignmentConfirmationEffects{ActorID: actor})
		}(subject)
	}
	close(start)
	require.NoError(t, <-results)
	require.NoError(t, <-results)
	assertScalar(t, db, `SELECT COUNT(*) FROM assignments WHERE id=$1 AND status='finished' AND completed_at IS NOT NULL`, []any{id}, 1)
	assertScalar(t, db, `SELECT COUNT(*) FROM assignment_recipients WHERE assignment_id=$1 AND confirmed_at IS NOT NULL AND confirmed_by=$2`, []any{id, actor}, 2)
	assertScalar(t, db, `SELECT COUNT(*) FROM event_outbox WHERE deduplication_key LIKE $1`, []any{"assignment:" + id.String() + ":confirmed:%:journal"}, 2)
	require.ErrorIs(t, repo.ConfirmRecipientWithEffects(id, creator, models.AssignmentConfirmationEffects{ActorID: actor}), models.ErrAlreadyConfirmed)
	assertScalar(t, db, `SELECT COUNT(*) FROM event_outbox WHERE deduplication_key LIKE $1`, []any{"assignment:" + id.String() + ":confirmed:%:journal"}, 2)
}

func TestAssignmentAcknowledgmentContentLifecycleIntegration(t *testing.T) {
	db := integrationdb.Open(t)
	creator, document := seedIntegrationDocument(t, db)
	repo := NewAssignmentRepository(database.Wrap(db))
	id := uuid.New()
	now := time.Now().UTC()
	require.NoError(t, repo.CreateRecipientTaskWithOutbox(&models.Assignment{
		ID: id, DocumentID: document, CreatorID: creator, Content: "Ignored input", CreatedAt: now,
		Users: []models.AssignmentRecipient{{ID: uuid.New(), UserID: creator, CreatedAt: now}},
	}, nil))
	var stored string
	require.NoError(t, db.QueryRow(`SELECT content FROM assignments WHERE id=$1`, id).Scan(&stored))
	require.Empty(t, stored)
	execSQL(t, db, `UPDATE assignments SET content='legacy-ack-comment' WHERE id=$1`, id)
	due := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.UpdateRecipientTaskDeadline(id, &due, nil))
	require.NoError(t, db.QueryRow(`SELECT content FROM assignments WHERE id=$1`, id).Scan(&stored))
	require.Equal(t, "legacy-ack-comment", stored)
	item, err := repo.GetByID(id)
	require.NoError(t, err)
	require.Empty(t, item.Content)
	require.Equal(t, "2026-12-31", item.Deadline.Format("2006-01-02"))
	list, err := repo.GetList(models.AssignmentFilter{DocumentID: document.String(), AccessibleByUserID: creator.String(), Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Empty(t, list.Items[0].Content)
	list, err = repo.GetList(models.AssignmentFilter{Search: "legacy-ack-comment", DocumentID: document.String(), AccessibleByUserID: creator.String(), Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Empty(t, list.Items)
	require.Zero(t, list.TotalCount)
	execSQL(t, db, `INSERT INTO assignments (document_id, executor_id, content) VALUES ($1,$2,'legacy-ack-comment')`, document, creator)
	list, err = repo.GetList(models.AssignmentFilter{Search: "legacy-ack-comment", DocumentID: document.String(), AccessibleByUserID: creator.String(), Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, models.AssignmentTypeExecution, list.Items[0].Type)
	require.Equal(t, "legacy-ack-comment", list.Items[0].Content)
}

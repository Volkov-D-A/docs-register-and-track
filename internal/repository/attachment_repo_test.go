package repository

import (
	"context"
	"database/sql"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAttachmentRepo(t *testing.T) (*AttachmentRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	wrappedDB := &database.DB{DB: db}
	repo := NewAttachmentRepository(wrappedDB)
	return repo, mock
}

func TestAttachmentRepository_CreateWithOutbox(t *testing.T) {
	// Сохранение нового вложения в БД
	repo, mock := setupAttachmentRepo(t)

	attachment := &models.Attachment{
		DocumentID:  uuid.New(),
		Filename:    "test.txt",
		StoragePath: "minio/path",
		FileSize:    11,
		ContentType: "text/plain",
		UploadedBy:  uuid.New(),
	}

	expectedID := uuid.New()
	expectedUploadedAt := time.Now()

	t.Run("success", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO attachments \(document_id, filename, storage_path, file_size, content_type, uploaded_by\)`).
			WithArgs(attachment.DocumentID, attachment.Filename, attachment.StoragePath, attachment.FileSize, attachment.ContentType, attachment.UploadedBy).
			WillReturnRows(sqlmock.NewRows([]string{"id", "uploaded_at"}).AddRow(expectedID, expectedUploadedAt))
		mock.ExpectExec(`UPDATE storage_statistics`).WithArgs(attachment.FileSize).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.CreateWithOutbox(attachment, nil)

		require.NoError(t, err)
		assert.Equal(t, expectedID, attachment.ID)
		assert.Equal(t, expectedUploadedAt, attachment.UploadedAt)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttachmentRepositoryCreateForAssignmentRejectsNonCurrentIteration(t *testing.T) {
	repo, mockDB := setupAttachmentRepo(t)
	assignmentID := uuid.New()
	attachment := &models.Attachment{
		DocumentID:   uuid.New(),
		AssignmentID: &assignmentID,
		Filename:     "result.pdf",
		StoragePath:  "series/result.pdf",
		FileSize:     42,
		ContentType:  "application/pdf",
		UploadedBy:   uuid.New(),
	}
	query := `
		INSERT INTO attachments (document_id, assignment_id, filename, storage_path, file_size, content_type, uploaded_by)
		SELECT $1,$2,$3,$4,$5,$6,$7
		FROM assignments assignment
		LEFT JOIN assignment_series series ON series.id=assignment.series_id
		WHERE assignment.id=$2
		  AND assignment.document_id=$1
		  AND (assignment.series_id IS NULL OR series.current_assignment_id=assignment.id)
		  AND (NOT $8 OR assignment.status='in_progress')
		RETURNING id,uploaded_at
	`

	mockDB.ExpectBegin()
	mockDB.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(
		attachment.DocumentID, assignmentID, attachment.Filename, attachment.StoragePath,
		attachment.FileSize, attachment.ContentType, attachment.UploadedBy, true,
	).WillReturnError(sql.ErrNoRows)
	mockDB.ExpectRollback()

	err := repo.CreateForAssignmentWithOutbox(attachment, true, nil)
	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, "CONFLICT", appErr.Kind)
	assert.Equal(t, 409, appErr.Code)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestAttachmentRepositoryCreateForAssignmentPersistsEligibleIteration(t *testing.T) {
	repo, mockDB := setupAttachmentRepo(t)
	assignmentID := uuid.New()
	attachment := &models.Attachment{
		DocumentID:   uuid.New(),
		AssignmentID: &assignmentID,
		Filename:     "result.pdf",
		StoragePath:  "series/result.pdf",
		FileSize:     42,
		ContentType:  "application/pdf",
		UploadedBy:   uuid.New(),
	}
	expectedID, expectedUploadedAt := uuid.New(), time.Now().UTC()

	mockDB.ExpectBegin()
	mockDB.ExpectQuery(`INSERT INTO attachments`).WithArgs(
		attachment.DocumentID, assignmentID, attachment.Filename, attachment.StoragePath,
		attachment.FileSize, attachment.ContentType, attachment.UploadedBy, true,
	).WillReturnRows(sqlmock.NewRows([]string{"id", "uploaded_at"}).AddRow(expectedID, expectedUploadedAt))
	mockDB.ExpectExec(`UPDATE storage_statistics`).WithArgs(attachment.FileSize).WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectCommit()

	require.NoError(t, repo.CreateForAssignmentWithOutbox(attachment, true, nil))
	assert.Equal(t, expectedID, attachment.ID)
	assert.Equal(t, expectedUploadedAt, attachment.UploadedAt)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestAttachmentRepositoryDeleteMarkedAndDecrementStorageStatistics(t *testing.T) {
	repo, mock := setupAttachmentRepo(t)
	attachmentID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT file_size FROM attachments WHERE id = \$1 AND deletion_requested_at IS NOT NULL FOR UPDATE`).
		WithArgs(attachmentID).
		WillReturnRows(sqlmock.NewRows([]string{"file_size"}).AddRow(42))
	mock.ExpectExec(`DELETE FROM attachments WHERE id = \$1 AND deletion_requested_at IS NOT NULL`).
		WithArgs(attachmentID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE storage_statistics`).WithArgs(42).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.DeleteMarkedAndDecrementStorageStatistics(attachmentID))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAttachmentRepositoryCreateWithOutboxRollsBackOnEnqueueFailure(t *testing.T) {
	repo, mock := setupAttachmentRepo(t)
	repo.SetOutbox(NewOutboxRepository(repo.db))
	attachment := &models.Attachment{DocumentID: uuid.New(), Filename: "test.txt", StoragePath: "objects/test.txt", FileSize: 1, ContentType: "text/plain", UploadedBy: uuid.New()}
	event := models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "attachment:test:upload:journal", Payload: `{}`}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO attachments`).WithArgs(attachment.DocumentID, attachment.Filename, attachment.StoragePath, attachment.FileSize, attachment.ContentType, attachment.UploadedBy).WillReturnRows(sqlmock.NewRows([]string{"id", "uploaded_at"}).AddRow(uuid.New(), time.Now()))
	mock.ExpectExec(`UPDATE storage_statistics`).WithArgs(attachment.FileSize).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs(event.EventType, event.DeduplicationKey, event.Payload).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	require.ErrorIs(t, repo.CreateWithOutbox(attachment, []models.OutboxEvent{event}), assert.AnError)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAttachmentRepositoryMarkDeletingWithOutboxRequiresOutbox(t *testing.T) {
	repo, _ := setupAttachmentRepo(t)
	err := repo.MarkDeletingWithOutbox(models.Attachment{ID: uuid.New(), StoragePath: "objects/test.txt"})
	require.ErrorIs(t, err, ErrOutboxNotConfigured)
}

func TestAttachmentRepository_DeletionSaga(t *testing.T) {
	repo, mock := setupAttachmentRepo(t)
	attachmentID := uuid.New()

	t.Run("atomically marks and queues deletion", func(t *testing.T) {
		outboxRepo := NewOutboxRepository(repo.db)
		repo.SetOutbox(outboxRepo)
		attachment := models.Attachment{ID: attachmentID, StoragePath: "objects/test.txt"}
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE attachments SET deletion_requested_at = CURRENT_TIMESTAMP`).WithArgs(attachmentID).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs(models.OutboxEventFileDelete, "attachment:"+attachmentID.String()+":delete", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		require.NoError(t, repo.MarkDeletingWithOutbox(attachment))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttachmentRepository_GetByID(t *testing.T) {
	// Получение метаданных вложения по ID
	repo, mock := setupAttachmentRepo(t)
	attachmentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, document_id, filename, storage_path, file_size, content_type, uploaded_by, uploaded_at FROM attachments WHERE id = \$1 AND deletion_requested_at IS NULL`).
			WithArgs(attachmentID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "filename", "storage_path", "file_size", "content_type", "uploaded_by", "uploaded_at"}).
				AddRow(attachmentID, uuid.New(), "test.txt", "minio-path", 11, "text/plain", uuid.New(), time.Now()))

		attachment, err := repo.GetByID(attachmentID)

		require.NoError(t, err)
		require.NotNil(t, attachment)
		assert.Equal(t, attachmentID, attachment.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, document_id, filename, storage_path, file_size, content_type, uploaded_by, uploaded_at FROM attachments WHERE id = \$1 AND deletion_requested_at IS NULL`).
			WithArgs(attachmentID).
			WillReturnError(sql.ErrNoRows)

		attachment, err := repo.GetByID(attachmentID)

		require.NoError(t, err)
		assert.Nil(t, attachment)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttachmentRepository_GetByDocumentID(t *testing.T) {
	// Получение списка вложений для определенного документа
	repo, mock := setupAttachmentRepo(t)
	docID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT a.id, a.document_id, a.filename, a.file_size, a.content_type, a.storage_path, a.uploaded_by, a.uploaded_at, u.full_name FROM attachments a LEFT JOIN users u ON a.uploaded_by = u.id WHERE a.document_id = \$1 AND a.deletion_requested_at IS NULL ORDER BY a.uploaded_at DESC`).
			WithArgs(docID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "filename", "file_size", "content_type", "storage_path", "uploaded_by", "uploaded_at", "full_name"}).
				AddRow(uuid.New(), docID, "test1.txt", 11, "text/plain", "path1", uuid.New(), time.Now(), "User One").
				AddRow(uuid.New(), docID, "test2.pdf", 42, "application/pdf", "path2", uuid.New(), time.Now(), "User Two"))

		attachments, err := repo.GetByDocumentID(docID)

		require.NoError(t, err)
		require.Len(t, attachments, 2)
		assert.Equal(t, "User One", attachments[0].UploadedByName)
		assert.Equal(t, "User Two", attachments[1].UploadedByName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no attachments", func(t *testing.T) {
		mock.ExpectQuery(`SELECT a.id, a.document_id, a.filename, a.file_size, a.content_type, a.storage_path, a.uploaded_by, a.uploaded_at, u.full_name FROM attachments a LEFT JOIN users u ON a.uploaded_by = u.id WHERE a.document_id = \$1 AND a.deletion_requested_at IS NULL ORDER BY a.uploaded_at DESC`).
			WithArgs(docID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "filename", "file_size", "content_type", "storage_path", "uploaded_by", "uploaded_at", "full_name"}))

		attachments, err := repo.GetByDocumentID(docID)

		require.NoError(t, err)
		require.Empty(t, attachments)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttachmentRepository_GetOlderThan(t *testing.T) {
	repo, mock := setupAttachmentRepo(t)
	cutoff := time.Now().Add(-24 * time.Hour)
	attachmentID := uuid.New()
	docID := uuid.New()
	uploaderID := uuid.New()
	uploadedAt := cutoff.Add(-time.Hour)

	mock.ExpectQuery(`SELECT id, document_id, filename, storage_path, file_size, content_type, uploaded_by, uploaded_at FROM attachments WHERE uploaded_at < \$1 AND deletion_requested_at IS NULL`).
		WithArgs(cutoff).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "filename", "storage_path", "file_size", "content_type", "uploaded_by", "uploaded_at",
		}).AddRow(attachmentID, docID, "old.pdf", "path/old.pdf", 42, "application/pdf", uploaderID, uploadedAt))

	attachments, err := repo.GetOlderThan(cutoff)

	require.NoError(t, err)
	require.Len(t, attachments, 1)
	assert.Equal(t, attachmentID, attachments[0].ID)
	assert.Equal(t, "old.pdf", attachments[0].Filename)
	assert.Equal(t, uploadedAt, attachments[0].UploadedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAttachmentRepository_MarkDeletingMultipleWithOutbox(t *testing.T) {
	repo, mock := setupAttachmentRepo(t)

	t.Run("empty input", func(t *testing.T) {
		err := repo.MarkDeletingMultipleWithOutbox(nil, nil)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		repo.SetOutbox(NewOutboxRepository(repo.db))
		attachments := []models.Attachment{
			{ID: uuid.New(), StoragePath: "objects/first.pdf"},
			{ID: uuid.New(), StoragePath: "objects/second.pdf"},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE attachments SET deletion_requested_at = CURRENT_TIMESTAMP WHERE id = ANY\(\$1\) AND deletion_requested_at IS NULL`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		for _, attachment := range attachments {
			mock.ExpectExec(`INSERT INTO event_outbox`).
				WithArgs(models.OutboxEventFileDelete, "attachment:"+attachment.ID.String()+":delete", sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(1, 1))
		}
		mock.ExpectCommit()

		err := repo.MarkDeletingMultipleWithOutbox(attachments, nil)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttachmentRepositoryStorageMutationInvalidatesRefresh(t *testing.T) {
	repo, mock := setupAttachmentRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM storage_statistics WHERE id = true FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(true))
	mock.ExpectExec(`DELETE FROM storage_statistics_mutations WHERE lease_until < CURRENT_TIMESTAMP`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO storage_statistics_mutations`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE storage_statistics`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mutation, err := repo.BeginStorageMutation(context.Background())
	require.NoError(t, err)
	require.NotNil(t, mutation.Context())

	mock.ExpectExec(`DELETE FROM storage_statistics_mutations WHERE token = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, mutation.Finish())
	require.NoError(t, mock.ExpectationsWereMet())
}

package repository

import (
	"database/sql"
	"docflow/internal/database"
	"docflow/internal/models"
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

func TestAttachmentRepository_Create(t *testing.T) {
	// Сохранение нового вложения в БД
	repo, mock := setupAttachmentRepo(t)

	attachment := &models.Attachment{
		DocumentID:   uuid.New(),
		DocumentType: "incoming",
		Filename:     "test.txt",
		StoragePath:  "minio/path",
		FileSize:     11,
		ContentType:  "text/plain",
		UploadedBy:   uuid.New(),
	}

	expectedID := uuid.New()
	expectedUploadedAt := time.Now()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO attachments \(document_id, document_type, filename, storage_path, file_size, content_type, uploaded_by\)`).
			WithArgs(attachment.DocumentID, attachment.DocumentType, attachment.Filename, attachment.StoragePath, attachment.FileSize, attachment.ContentType, attachment.UploadedBy).
			WillReturnRows(sqlmock.NewRows([]string{"id", "uploaded_at"}).AddRow(expectedID, expectedUploadedAt))

		err := repo.Create(attachment)

		require.NoError(t, err)
		assert.Equal(t, expectedID, attachment.ID)
		assert.Equal(t, expectedUploadedAt, attachment.UploadedAt)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttachmentRepository_Delete(t *testing.T) {
	// Удаление вложения по его ID
	repo, mock := setupAttachmentRepo(t)
	attachmentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM attachments WHERE id = \$1`).
			WithArgs(attachmentID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Delete(attachmentID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttachmentRepository_GetByID(t *testing.T) {
	// Получение метаданных вложения по ID
	repo, mock := setupAttachmentRepo(t)
	attachmentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, document_id, document_type, filename, storage_path, file_size, content_type, uploaded_by, uploaded_at FROM attachments WHERE id = \$1`).
			WithArgs(attachmentID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "document_type", "filename", "storage_path", "file_size", "content_type", "uploaded_by", "uploaded_at"}).
				AddRow(attachmentID, uuid.New(), "incoming", "test.txt", "minio-path", 11, "text/plain", uuid.New(), time.Now()))

		attachment, err := repo.GetByID(attachmentID)

		require.NoError(t, err)
		require.NotNil(t, attachment)
		assert.Equal(t, attachmentID, attachment.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, document_id, document_type, filename, storage_path, file_size, content_type, uploaded_by, uploaded_at FROM attachments WHERE id = \$1`).
			WithArgs(attachmentID).
			WillReturnError(sql.ErrNoRows)

		attachment, err := repo.GetByID(attachmentID)

		require.Error(t, err)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.Nil(t, attachment)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAttachmentRepository_GetByDocumentID(t *testing.T) {
	// Получение списка вложений для определенного документа
	repo, mock := setupAttachmentRepo(t)
	docID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT a.id, a.document_id, a.document_type, a.filename, a.file_size, a.content_type, a.storage_path, a.uploaded_by, a.uploaded_at, u.full_name FROM attachments a LEFT JOIN users u ON a.uploaded_by = u.id WHERE a.document_id = \$1 ORDER BY a.uploaded_at DESC`).
			WithArgs(docID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "document_type", "filename", "file_size", "content_type", "storage_path", "uploaded_by", "uploaded_at", "full_name"}).
				AddRow(uuid.New(), docID, "incoming", "test1.txt", 11, "text/plain", "path1", uuid.New(), time.Now(), "User One").
				AddRow(uuid.New(), docID, "incoming", "test2.pdf", 42, "application/pdf", "path2", uuid.New(), time.Now(), "User Two"))

		attachments, err := repo.GetByDocumentID(docID)

		require.NoError(t, err)
		require.Len(t, attachments, 2)
		assert.Equal(t, "User One", attachments[0].UploadedByName)
		assert.Equal(t, "User Two", attachments[1].UploadedByName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no attachments", func(t *testing.T) {
		mock.ExpectQuery(`SELECT a.id, a.document_id, a.document_type, a.filename, a.file_size, a.content_type, a.storage_path, a.uploaded_by, a.uploaded_at, u.full_name FROM attachments a LEFT JOIN users u ON a.uploaded_by = u.id WHERE a.document_id = \$1 ORDER BY a.uploaded_at DESC`).
			WithArgs(docID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "document_type", "filename", "file_size", "content_type", "storage_path", "uploaded_by", "uploaded_at", "full_name"}))

		attachments, err := repo.GetByDocumentID(docID)

		require.NoError(t, err)
		require.Empty(t, attachments)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

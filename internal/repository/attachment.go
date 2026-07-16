package repository

import (
	"database/sql"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// AttachmentRepository предоставляет методы для работы с вложениями (файлами) в БД.
type AttachmentRepository struct {
	db *database.DB
}

// NewAttachmentRepository создает новый экземпляр AttachmentRepository.
func NewAttachmentRepository(db *database.DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

// Create сохраняет новое вложение в БД.
func (r *AttachmentRepository) Create(a *models.Attachment) error {
	return r.db.QueryRow(
		`INSERT INTO attachments (document_id, filename, storage_path, file_size, content_type, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, uploaded_at`,
		a.DocumentID, a.Filename, a.StoragePath, a.FileSize, a.ContentType, a.UploadedBy,
	).Scan(&a.ID, &a.UploadedAt)
}

// MarkDeleting durable records the intent to delete before the object is
// removed from external storage. The row becomes invisible to regular reads.
func (r *AttachmentRepository) MarkDeleting(id uuid.UUID) error {
	_, err := r.db.Exec(`UPDATE attachments
		SET deletion_requested_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND deletion_requested_at IS NULL`, id)
	return err
}

// MarkDeletingMultiple atomically marks a batch before any MinIO operation.
func (r *AttachmentRepository) MarkDeletingMultiple(ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(`UPDATE attachments
		SET deletion_requested_at = CURRENT_TIMESTAMP
		WHERE id = ANY($1) AND deletion_requested_at IS NULL`, pq.Array(ids))
	return err
}

// DeleteMarked removes only an attachment whose durable deletion intent was
// committed. A failed database delete therefore leaves a retryable tombstone.
func (r *AttachmentRepository) DeleteMarked(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM attachments WHERE id = $1 AND deletion_requested_at IS NOT NULL", id)
	return err
}

// GetByID возвращает метаданные вложения (без контента) по ID.
func (r *AttachmentRepository) GetByID(id uuid.UUID) (*models.Attachment, error) {
	var a models.Attachment
	if err := r.db.QueryRow(
		`SELECT id, document_id, filename, storage_path, file_size, content_type, uploaded_by, uploaded_at
		FROM attachments WHERE id = $1 AND deletion_requested_at IS NULL`,
		id,
	).Scan(&a.ID, &a.DocumentID, &a.Filename, &a.StoragePath, &a.FileSize, &a.ContentType, &a.UploadedBy, &a.UploadedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

// GetByDocumentID возвращает все вложения, прикрепленные к определенному документу.
func (r *AttachmentRepository) GetByDocumentID(docID uuid.UUID) ([]models.Attachment, error) {
	rows, err := r.db.Query(
		`SELECT a.id, a.document_id, a.filename, a.file_size, a.content_type, a.storage_path, a.uploaded_by, a.uploaded_at, u.full_name
		FROM attachments a
		LEFT JOIN users u ON a.uploaded_by = u.id
		WHERE a.document_id = $1 AND a.deletion_requested_at IS NULL
		ORDER BY a.uploaded_at DESC`,
		docID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make([]models.Attachment, 0)
	for rows.Next() {
		var a models.Attachment
		var uploadedByName sql.NullString
		if err := rows.Scan(
			&a.ID, &a.DocumentID, &a.Filename, &a.FileSize, &a.ContentType, &a.StoragePath, &a.UploadedBy, &a.UploadedAt, &uploadedByName,
		); err != nil {
			return nil, err
		}
		a.UploadedByName = uploadedByName.String

		attachments = append(attachments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attachments, nil
}

// GetOlderThan возвращает вложения, загруженные до указанной даты.
func (r *AttachmentRepository) GetOlderThan(date time.Time) ([]models.Attachment, error) {
	rows, err := r.db.Query(
		`SELECT id, document_id, filename, storage_path, file_size, content_type, uploaded_by, uploaded_at
		FROM attachments WHERE uploaded_at < $1 AND deletion_requested_at IS NULL`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make([]models.Attachment, 0)
	for rows.Next() {
		var a models.Attachment
		if err := rows.Scan(
			&a.ID, &a.DocumentID, &a.Filename, &a.StoragePath, &a.FileSize, &a.ContentType, &a.UploadedBy, &a.UploadedAt,
		); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attachments, nil
}

// GetPendingDeletion returns hidden attachments whose object deletion must be
// retried after an interrupted or failed operation.
func (r *AttachmentRepository) GetPendingDeletion() ([]models.Attachment, error) {
	rows, err := r.db.Query(`SELECT id, document_id, filename, storage_path, file_size, content_type, uploaded_by, uploaded_at
		FROM attachments WHERE deletion_requested_at IS NOT NULL
		ORDER BY deletion_requested_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make([]models.Attachment, 0)
	for rows.Next() {
		var a models.Attachment
		if err := rows.Scan(&a.ID, &a.DocumentID, &a.Filename, &a.StoragePath, &a.FileSize, &a.ContentType, &a.UploadedBy, &a.UploadedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

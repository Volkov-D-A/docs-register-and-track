package repository

import (
	"database/sql"
	"docflow/internal/database"
	"docflow/internal/models"

	"github.com/google/uuid"
)

type AttachmentRepository struct {
	db *database.DB
}

func NewAttachmentRepository(db *database.DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

func (r *AttachmentRepository) Create(a *models.Attachment) error {
	return r.db.QueryRow(
		`INSERT INTO attachments (document_id, document_type, filename, content, file_size, content_type, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, uploaded_at`,
		a.DocumentID, a.DocumentType, a.Filename, a.Content, a.FileSize, a.ContentType, a.UploadedBy,
	).Scan(&a.ID, &a.UploadedAt)
}

func (r *AttachmentRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM attachments WHERE id = $1", id)
	return err
}

func (r *AttachmentRepository) GetByID(id uuid.UUID) (*models.Attachment, error) {
	var a models.Attachment
	if err := r.db.QueryRow(
		`SELECT id, document_id, document_type, filename, file_size, content_type, uploaded_by, uploaded_at
		FROM attachments WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.DocumentID, &a.DocumentType, &a.Filename, &a.FileSize, &a.ContentType, &a.UploadedBy, &a.UploadedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AttachmentRepository) GetByDocumentID(docID uuid.UUID) ([]models.Attachment, error) {
	rows, err := r.db.Query(
		`SELECT a.id, a.document_id, a.document_type, a.filename, a.file_size, a.content_type, a.uploaded_by, a.uploaded_at, u.full_name
		FROM attachments a
		LEFT JOIN users u ON a.uploaded_by = u.id
		WHERE a.document_id = $1
		ORDER BY a.uploaded_at DESC`,
		docID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []models.Attachment
	for rows.Next() {
		var a models.Attachment
		var uploadedByName sql.NullString
		if err := rows.Scan(
			&a.ID, &a.DocumentID, &a.DocumentType, &a.Filename, &a.FileSize, &a.ContentType, &a.UploadedBy, &a.UploadedAt, &uploadedByName,
		); err != nil {
			return nil, err
		}
		a.UploadedByName = uploadedByName.String

		attachments = append(attachments, a)
	}
	return attachments, nil
}

func (r *AttachmentRepository) GetContent(id uuid.UUID) ([]byte, error) {
	var content []byte
	err := r.db.QueryRow("SELECT content FROM attachments WHERE id = $1", id).Scan(&content)
	if err != nil {
		return nil, err
	}
	return content, nil
}

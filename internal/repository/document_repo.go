package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentRepository предоставляет доступ к общей корневой сущности документа.
type DocumentRepository struct {
	db *database.DB
}

// NewDocumentRepository создает новый экземпляр DocumentRepository.
func NewDocumentRepository(db *database.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// GetByID возвращает общий документ по ID.
func (r *DocumentRepository) GetByID(id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	err := r.db.QueryRow(`
		SELECT id, kind, nomenclature_id, registration_number, registration_date, document_type_id, content, pages_count, created_by, created_at, updated_at
		FROM documents
		WHERE id = $1
	`, id).Scan(
		&doc.ID,
		&doc.Kind,
		&doc.NomenclatureID,
		&doc.RegistrationNumber,
		&doc.RegistrationDate,
		&doc.DocumentTypeID,
		&doc.Content,
		&doc.PagesCount,
		&doc.CreatedBy,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	return &doc, nil
}

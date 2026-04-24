package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

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

// GetByIDs возвращает общие документы по списку ID.
func (r *DocumentRepository) GetByIDs(ids []uuid.UUID) ([]models.Document, error) {
	if len(ids) == 0 {
		return []models.Document{}, nil
	}

	idStrings := make([]string, 0, len(ids))
	for _, id := range ids {
		idStrings = append(idStrings, id.String())
	}

	rows, err := r.db.Query(`
		SELECT id, kind, nomenclature_id, registration_number, registration_date, document_type_id, content, pages_count, created_by, created_at, updated_at
		FROM documents
		WHERE id = ANY($1::uuid[])
	`, pq.Array(idStrings))
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}
	defer rows.Close()

	docs := make([]models.Document, 0, len(ids))
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return docs, nil
}

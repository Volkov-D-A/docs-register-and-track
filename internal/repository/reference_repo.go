package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"docflow/internal/database"
	"docflow/internal/models"
)

// ReferenceRepository предоставляет методы для работы со справочниками (типы документов, организации) в БД.
type ReferenceRepository struct {
	db *database.DB
}

// NewReferenceRepository создает новый экземпляр ReferenceRepository.
func NewReferenceRepository(db *database.DB) *ReferenceRepository {
	return &ReferenceRepository{db: db}
}

// === Типы документов ===

// GetAllDocumentTypes возвращает все типы документов.
func (r *ReferenceRepository) GetAllDocumentTypes() ([]models.DocumentType, error) {
	rows, err := r.db.Query(`
		SELECT id, name, created_at FROM document_types ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get document types: %w", err)
	}
	defer rows.Close()

	items := make([]models.DocumentType, 0)
	for rows.Next() {
		var item models.DocumentType
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// CreateDocumentType создает новый тип документа.
func (r *ReferenceRepository) CreateDocumentType(name string) (*models.DocumentType, error) {
	var id uuid.UUID
	err := r.db.QueryRow(`
		INSERT INTO document_types (name) VALUES ($1) RETURNING id
	`, name).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create document type: %w", err)
	}

	item := &models.DocumentType{}
	err = r.db.QueryRow(`
		SELECT id, name, created_at FROM document_types WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// UpdateDocumentType обновляет тип документа.
func (r *ReferenceRepository) UpdateDocumentType(id uuid.UUID, name string) error {
	_, err := r.db.Exec(`UPDATE document_types SET name = $1 WHERE id = $2`, name, id)
	if err != nil {
		return fmt.Errorf("failed to update document type: %w", err)
	}
	return nil
}

// DeleteDocumentType удаляет тип документа.
func (r *ReferenceRepository) DeleteDocumentType(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM document_types WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete document type: %w", err)
	}
	return nil
}

// === Организации ===

// GetAllOrganizations возвращает все организации-корреспонденты.
func (r *ReferenceRepository) GetAllOrganizations() ([]models.Organization, error) {
	rows, err := r.db.Query(`
		SELECT id, name, created_at FROM organizations ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}
	defer rows.Close()

	items := make([]models.Organization, 0)
	for rows.Next() {
		var item models.Organization
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// FindOrCreateOrganization ищет организацию по имени и возвращает её ID, если не найдена - создает новую.
func (r *ReferenceRepository) FindOrCreateOrganization(name string) (*models.Organization, error) {
	// Сначала ищем существующую
	var item models.Organization
	err := r.db.QueryRow(`
		SELECT id, name, created_at FROM organizations WHERE name = $1
	`, name).Scan(&item.ID, &item.Name, &item.CreatedAt)

	if err == nil {

		return &item, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to find organization: %w", err)
	}

	// Создаём новую
	var id uuid.UUID
	err = r.db.QueryRow(`
		INSERT INTO organizations (name) VALUES ($1) RETURNING id
	`, name).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	err = r.db.QueryRow(`
		SELECT id, name, created_at FROM organizations WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// SearchOrganizations выполняет поиск организаций по названию.
func (r *ReferenceRepository) SearchOrganizations(query string) ([]models.Organization, error) {
	rows, err := r.db.Query(`
		SELECT id, name, created_at FROM organizations
		WHERE name ILIKE $1
		ORDER BY name LIMIT 20
	`, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search organizations: %w", err)
	}
	defer rows.Close()

	items := make([]models.Organization, 0)
	for rows.Next() {
		var item models.Organization
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// UpdateOrganization обновляет название организации.
func (r *ReferenceRepository) UpdateOrganization(id uuid.UUID, name string) error {
	_, err := r.db.Exec(`UPDATE organizations SET name = $1 WHERE id = $2`, name, id)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}
	return nil
}

// DeleteOrganization удаляет организацию.
func (r *ReferenceRepository) DeleteOrganization(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	return nil
}

package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

// ReferenceRepository предоставляет методы для работы со справочниками организаций и исполнителей резолюции в БД.
type ReferenceRepository struct {
	db     *database.DB
	outbox *OutboxRepository
}

func (r *ReferenceRepository) SetOutbox(outbox *OutboxRepository) { r.outbox = outbox }

func (r *ReferenceRepository) execWithOutbox(query string, args []interface{}, effects []models.OutboxEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return err
	}
	return tx.Commit()
}

// NewReferenceRepository создает новый экземпляр ReferenceRepository.
func NewReferenceRepository(db *database.DB) *ReferenceRepository {
	return &ReferenceRepository{db: db}
}

// === Организации ===

// GetAllOrganizations возвращает все организации-корреспонденты.
func (r *ReferenceRepository) GetAllOrganizations() ([]models.Organization, error) {
	rows, err := r.db.Query(`
		SELECT id, name FROM organizations ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}
	defer rows.Close()

	items := make([]models.Organization, 0)
	for rows.Next() {
		var item models.Organization
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
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
		SELECT id, name FROM organizations WHERE name = $1
	`, name).Scan(&item.ID, &item.Name)

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

	return &models.Organization{ID: id, Name: name}, nil
}

// SearchOrganizations выполняет поиск организаций по названию.
func (r *ReferenceRepository) SearchOrganizations(query string) ([]models.Organization, error) {
	rows, err := r.db.Query(`
		SELECT id, name FROM organizations
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
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ReferenceRepository) UpdateOrganizationWithOutbox(id uuid.UUID, name string, effects []models.OutboxEvent) error {
	return r.execWithOutbox(`UPDATE organizations SET name = $1 WHERE id = $2`, []interface{}{name, id}, effects)
}

func (r *ReferenceRepository) DeleteOrganizationWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error {
	return r.execWithOutbox(`DELETE FROM organizations WHERE id = $1`, []interface{}{id}, effects)
}

func (r *ReferenceRepository) MergeOrganizationsWithOutbox(sourceID, targetID uuid.UUID, effects []models.OutboxEvent) error {
	if sourceID == targetID {
		return models.NewBadRequest("нельзя объединить организацию саму с собой")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin organization merge transaction: %w", err)
	}
	defer tx.Rollback()
	var name string
	if err := tx.QueryRow(`SELECT name FROM organizations WHERE id = $1 FOR UPDATE`, sourceID).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return models.NewBadRequest("исходная организация не найдена")
		}
		return err
	}
	if err := tx.QueryRow(`SELECT name FROM organizations WHERE id = $1 FOR UPDATE`, targetID).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return models.NewBadRequest("целевая организация не найдена")
		}
		return err
	}
	if _, err := tx.Exec(`UPDATE document_correspondent_registrations SET correspondent_org_id = $1 WHERE correspondent_org_id = $2`, targetID, sourceID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE outgoing_document_details SET recipient_org_id = $1 WHERE recipient_org_id = $2`, targetID, sourceID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM organizations WHERE id = $1`, sourceID); err != nil {
		return err
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return err
	}
	return tx.Commit()
}

// === Исполнители резолюции ===

// GetAllResolutionExecutors возвращает всех исполнителей резолюции.
func (r *ReferenceRepository) GetAllResolutionExecutors() ([]models.ResolutionExecutor, error) {
	rows, err := r.db.Query(`
		SELECT id, name FROM resolution_executors ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolution executors: %w", err)
	}
	defer rows.Close()

	items := make([]models.ResolutionExecutor, 0)
	for rows.Next() {
		var item models.ResolutionExecutor
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// FindOrCreateResolutionExecutor ищет исполнителя по имени и возвращает его, если не найден — создает нового.
func (r *ReferenceRepository) FindOrCreateResolutionExecutor(name string) (*models.ResolutionExecutor, error) {
	var item models.ResolutionExecutor
	err := r.db.QueryRow(`
		SELECT id, name FROM resolution_executors WHERE name = $1
	`, name).Scan(&item.ID, &item.Name)

	if err == nil {
		return &item, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to find resolution executor: %w", err)
	}

	// Создаём нового
	var id uuid.UUID
	err = r.db.QueryRow(`
		INSERT INTO resolution_executors (name) VALUES ($1) RETURNING id
	`, name).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolution executor: %w", err)
	}

	return &models.ResolutionExecutor{ID: id, Name: name}, nil
}

// SearchResolutionExecutors выполняет поиск исполнителей резолюции по имени.
func (r *ReferenceRepository) SearchResolutionExecutors(query string) ([]models.ResolutionExecutor, error) {
	rows, err := r.db.Query(`
		SELECT id, name FROM resolution_executors
		WHERE name ILIKE $1
		ORDER BY name LIMIT 20
	`, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search resolution executors: %w", err)
	}
	defer rows.Close()

	items := make([]models.ResolutionExecutor, 0)
	for rows.Next() {
		var item models.ResolutionExecutor
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ReferenceRepository) UpdateResolutionExecutorWithOutbox(id uuid.UUID, name string, effects []models.OutboxEvent) error {
	return r.execWithOutbox(`UPDATE resolution_executors SET name = $1 WHERE id = $2`, []interface{}{name, id}, effects)
}

func (r *ReferenceRepository) DeleteResolutionExecutorWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error {
	return r.execWithOutbox(`DELETE FROM resolution_executors WHERE id = $1`, []interface{}{id}, effects)
}

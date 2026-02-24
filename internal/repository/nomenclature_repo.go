package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/database"
	"docflow/internal/models"
)

type NomenclatureRepository struct {
	db *database.DB
}

func NewNomenclatureRepository(db *database.DB) *NomenclatureRepository {
	return &NomenclatureRepository{db: db}
}

func (r *NomenclatureRepository) GetAll(year int, direction string) ([]models.Nomenclature, error) {
	query := `SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if year > 0 {
		query += fmt.Sprintf(" AND year = $%d", argIdx)
		args = append(args, year)
		argIdx++
	}
	if direction != "" {
		query += fmt.Sprintf(" AND direction = $%d", argIdx)
		args = append(args, direction)
		argIdx++
	}

	query += " ORDER BY index"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get nomenclature: %w", err)
	}
	defer rows.Close()

	var items []models.Nomenclature
	for rows.Next() {
		var item models.Nomenclature
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Index, &item.Year,
			&item.Direction, &item.NextNumber, &item.IsActive,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	return items, nil
}

func (r *NomenclatureRepository) GetByID(id uuid.UUID) (*models.Nomenclature, error) {
	item := &models.Nomenclature{}
	err := r.db.QueryRow(`
		SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature WHERE id = $1
	`, id).Scan(
		&item.ID, &item.Name, &item.Index, &item.Year,
		&item.Direction, &item.NextNumber, &item.IsActive,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get nomenclature by id: %w", err)
	}

	return item, nil
}

func (r *NomenclatureRepository) Create(name, index string, year int, direction string) (*models.Nomenclature, error) {
	var id uuid.UUID
	err := r.db.QueryRow(`
		INSERT INTO nomenclature (name, index, year, direction)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, name, index, year, direction).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create nomenclature: %w", err)
	}
	return r.GetByID(id)
}

func (r *NomenclatureRepository) Update(id uuid.UUID, name, index string, year int, direction string, isActive bool) (*models.Nomenclature, error) {
	_, err := r.db.Exec(`
		UPDATE nomenclature SET name = $1, index = $2, year = $3, direction = $4, is_active = $5, updated_at = $6
		WHERE id = $7
	`, name, index, year, direction, isActive, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to update nomenclature: %w", err)
	}
	return r.GetByID(id)
}

func (r *NomenclatureRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM nomenclature WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete nomenclature: %w", err)
	}
	return nil
}

// GetNextNumber — получить и инкрементировать следующий номер
func (r *NomenclatureRepository) GetNextNumber(id uuid.UUID) (int, string, error) {
	var nextNumber int
	var index string
	err := r.db.QueryRow(`
		UPDATE nomenclature SET next_number = next_number + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING next_number - 1, index
	`, id).Scan(&nextNumber, &index)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get next number: %w", err)
	}
	return nextNumber, index, nil
}

// GetActiveByDirection — активные дела по направлению
func (r *NomenclatureRepository) GetActiveByDirection(direction string, year int) ([]models.Nomenclature, error) {
	rows, err := r.db.Query(`
		SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature
		WHERE direction = $1 AND year = $2 AND is_active = true
		ORDER BY index
	`, direction, year)
	if err != nil {
		return nil, fmt.Errorf("failed to get active nomenclature: %w", err)
	}
	defer rows.Close()

	var items []models.Nomenclature
	for rows.Next() {
		var item models.Nomenclature
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Index, &item.Year,
			&item.Direction, &item.NextNumber, &item.IsActive,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	return items, nil
}

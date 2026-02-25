package repository

import (
	"database/sql"
	"docflow/internal/database"
	"docflow/internal/models"
	"fmt"

	"github.com/google/uuid"
)

type DepartmentRepository struct {
	db *database.DB
}

func NewDepartmentRepository(db *database.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

func (r *DepartmentRepository) GetAll() ([]models.Department, error) {
	query := `
		SELECT id, name, created_at, updated_at
		FROM departments
		ORDER BY name ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	departments := make([]models.Department, 0)
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}

		// Загружаем номенклатуру
		if err := r.loadNomenclature(&d); err != nil {
			return nil, err
		}

		departments = append(departments, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return departments, nil
}

func (r *DepartmentRepository) loadNomenclature(d *models.Department) error {
	query := `
		SELECT n.id, n.name, n.index, n.year, n.direction, n.next_number, n.is_active, n.created_at, n.updated_at
		FROM nomenclature n
		JOIN department_nomenclature dn ON n.id = dn.nomenclature_id
		WHERE dn.department_id = $1
		ORDER BY n.index
	`
	rows, err := r.db.Query(query, d.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var n models.Nomenclature
		if err := rows.Scan(
			&n.ID, &n.Name, &n.Index, &n.Year,
			&n.Direction, &n.NextNumber, &n.IsActive,
			&n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			return err
		}

		d.Nomenclature = append(d.Nomenclature, n)
		d.NomenclatureIDs = append(d.NomenclatureIDs, n.ID.String())
	}
	return rows.Err()
}

func (r *DepartmentRepository) GetNomenclatureIDs(departmentID uuid.UUID) ([]string, error) {
	query := `
		SELECT nomenclature_id
		FROM department_nomenclature
		WHERE department_id = $1
	`
	rows, err := r.db.Query(query, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id.String())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *DepartmentRepository) Create(name string, nomenclatureIDs []string) (*models.Department, error) {
	id := uuid.New()

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO departments (id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, name, created_at, updated_at
	`
	var d models.Department
	err = tx.QueryRow(query, id, name).Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if len(nomenclatureIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO department_nomenclature (department_id, nomenclature_id) VALUES ($1, $2)")
		if err != nil {
			return nil, err
		}
		defer stmt.Close()

		for _, nidStr := range nomenclatureIDs {
			nid, err := uuid.Parse(nidStr)
			if err != nil {
				return nil, fmt.Errorf("invalid nomenclature id %s: %w", nidStr, err)
			}
			if _, err := stmt.Exec(d.ID, nid); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Перечитываем чтобы вернуть полное состояние (хотя можно и так собрать)
	// Для простоты вернем то что есть, но с IDs
	d.NomenclatureIDs = nomenclatureIDs
	// models.Nomenclature не заполняем для оптимизации, если нужно - можно сделать Select
	return &d, nil
}

func (r *DepartmentRepository) Update(id uuid.UUID, name string, nomenclatureIDs []string) (*models.Department, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		UPDATE departments
		SET name = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, created_at, updated_at
	`
	var d models.Department
	err = tx.QueryRow(query, id, name).Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("department not found")
		}
		return nil, err
	}

	// Обновляем связи
	_, err = tx.Exec("DELETE FROM department_nomenclature WHERE department_id = $1", id)
	if err != nil {
		return nil, err
	}

	if len(nomenclatureIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO department_nomenclature (department_id, nomenclature_id) VALUES ($1, $2)")
		if err != nil {
			return nil, err
		}
		defer stmt.Close()

		for _, nidStr := range nomenclatureIDs {
			nid, err := uuid.Parse(nidStr)
			if err != nil {
				return nil, fmt.Errorf("invalid nomenclature id %s: %w", nidStr, err)
			}
			if _, err := stmt.Exec(d.ID, nid); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	d.NomenclatureIDs = nomenclatureIDs
	return &d, nil
}

func (r *DepartmentRepository) Delete(id uuid.UUID) error {
	query := `DELETE FROM departments WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("department not found")
	}
	return nil
}

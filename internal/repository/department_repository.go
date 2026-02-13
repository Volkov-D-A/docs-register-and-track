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

	var departments []models.Department
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.FillIDStr()
		departments = append(departments, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return departments, nil
}

func (r *DepartmentRepository) Create(name string) (*models.Department, error) {
	id := uuid.New()
	query := `
		INSERT INTO departments (id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, name, created_at, updated_at
	`
	var d models.Department
	err := r.db.QueryRow(query, id, name).Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.FillIDStr()
	return &d, nil
}

func (r *DepartmentRepository) Update(id uuid.UUID, name string) (*models.Department, error) {
	query := `
		UPDATE departments
		SET name = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, created_at, updated_at
	`
	var d models.Department
	err := r.db.QueryRow(query, id, name).Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("department not found")
		}
		return nil, err
	}
	d.FillIDStr()
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

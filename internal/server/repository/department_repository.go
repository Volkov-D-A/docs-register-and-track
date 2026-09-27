package repository

import (
	"database/sql"
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DepartmentRepository предоставляет методы для работы со справочником подразделений (отделов) в БД.
type DepartmentRepository struct {
	db     *database.DB
	outbox *OutboxRepository
}

func (r *DepartmentRepository) SetOutbox(outbox *OutboxRepository) { r.outbox = outbox }

// NewDepartmentRepository создает новый экземпляр DepartmentRepository.
func NewDepartmentRepository(db *database.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

// GetAll возвращает список подразделений и ID связанных с ними дел двумя запросами.
func (r *DepartmentRepository) GetAll() ([]models.Department, error) {
	rows, err := r.db.Query(`
		SELECT id, name
		FROM departments
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	departments := make([]models.Department, 0)
	departmentIDs := make([]uuid.UUID, 0)
	indexes := make(map[uuid.UUID]int)
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		indexes[d.ID] = len(departments)
		departmentIDs = append(departmentIDs, d.ID)
		departments = append(departments, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(departmentIDs) == 0 {
		return departments, nil
	}

	links, err := r.db.Query(`
		SELECT dn.department_id, dn.nomenclature_id
		FROM department_nomenclature dn
		JOIN nomenclature n ON n.id = dn.nomenclature_id
		WHERE dn.department_id = ANY($1)
		ORDER BY dn.department_id, n.index
	`, pq.Array(departmentIDs))
	if err != nil {
		return nil, err
	}
	defer links.Close()
	for links.Next() {
		var departmentID, nomenclatureID uuid.UUID
		if err := links.Scan(&departmentID, &nomenclatureID); err != nil {
			return nil, err
		}
		if index, ok := indexes[departmentID]; ok {
			departments[index].NomenclatureIDs = append(departments[index].NomenclatureIDs, nomenclatureID.String())
		}
	}
	if err := links.Err(); err != nil {
		return nil, err
	}
	return departments, nil
}

// GetNomenclatureIDs возвращает список ID номенклатур, привязанных к подразделению.
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

func (r *DepartmentRepository) CreateWithOutbox(name string, nomenclatureIDs []string, effects []models.OutboxEvent) (*models.Department, error) {
	return r.create(name, nomenclatureIDs, effects)
}

func (r *DepartmentRepository) create(name string, nomenclatureIDs []string, effects []models.OutboxEvent) (*models.Department, error) {
	id := uuid.New()

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO departments (id, name)
		VALUES ($1, $2)
		RETURNING id, name
	`
	var d models.Department
	err = tx.QueryRow(query, id, name).Scan(&d.ID, &d.Name)
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

	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	d.NomenclatureIDs = nomenclatureIDs
	return &d, nil
}

func (r *DepartmentRepository) UpdateWithOutbox(id uuid.UUID, name string, nomenclatureIDs []string, effects []models.OutboxEvent) (*models.Department, error) {
	return r.update(id, name, nomenclatureIDs, effects)
}

func (r *DepartmentRepository) update(id uuid.UUID, name string, nomenclatureIDs []string, effects []models.OutboxEvent) (*models.Department, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		UPDATE departments
		SET name = $2
		WHERE id = $1
		RETURNING id, name
	`
	var d models.Department
	err = tx.QueryRow(query, id, name).Scan(&d.ID, &d.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.NewNotFound("подразделение не найдено")
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

	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	d.NomenclatureIDs = nomenclatureIDs
	return &d, nil
}

func (r *DepartmentRepository) DeleteWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`DELETE FROM departments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return models.NewNotFound("подразделение не найдено")
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return err
	}
	return tx.Commit()
}

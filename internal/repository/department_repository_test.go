package repository

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"docflow/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepartmentRepository_GetAll(t *testing.T) {
	// Получение списка всех подразделений и связанных с ними номенклатур
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(&database.DB{DB: db})
	now := time.Now()
	depID := uuid.New()

	mock.ExpectQuery(`SELECT id, name, created_at, updated_at FROM departments ORDER BY name ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).AddRow(depID, "IT Отдел", now, now))

	nomQuery := `SELECT n.id, n.name, n.index, n.year, n.direction, n.next_number, n.is_active, n.created_at, n.updated_at
		FROM nomenclature n
		JOIN department_nomenclature dn ON n.id = dn.nomenclature_id
		WHERE dn.department_id = $1
		ORDER BY n.index`

	mock.ExpectQuery(regexp.QuoteMeta(nomQuery)).WithArgs(depID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "name", "index", "year", "direction", "next_number", "is_active", "created_at", "updated_at",
	}).AddRow(uuid.New(), "Дело", "01-01", 2024, "incoming", 1, true, now, now))

	deps, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "IT Отдел", deps[0].Name)
	assert.Len(t, deps[0].Nomenclature, 1)
	assert.Len(t, deps[0].NomenclatureIDs, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDepartmentRepository_GetNomenclatureIDs(t *testing.T) {
	// Получение массива идентификаторов номенклатур для подразделения
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(&database.DB{DB: db})
	depID := uuid.New()
	nomID := uuid.New()

	mock.ExpectQuery(`SELECT nomenclature_id FROM department_nomenclature WHERE department_id = \$1`).
		WithArgs(depID).WillReturnRows(sqlmock.NewRows([]string{"nomenclature_id"}).AddRow(nomID))

	ids, err := repo.GetNomenclatureIDs(depID)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Equal(t, nomID.String(), ids[0])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDepartmentRepository_Create(t *testing.T) {
	// Создание нового подразделения и привязка номенклатуры
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(&database.DB{DB: db})
	now := time.Now()
	nomID1 := uuid.New()

	mock.ExpectBegin()

	mock.ExpectQuery(`INSERT INTO departments \(id, name, created_at, updated_at\) VALUES \(\$1, \$2, NOW\(\), NOW\(\)\) RETURNING id, name, created_at, updated_at`).
		WithArgs(sqlmock.AnyArg(), "Новый Отдел").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).AddRow(uuid.New(), "Новый Отдел", now, now))

	mock.ExpectPrepare(`INSERT INTO department_nomenclature \(department_id, nomenclature_id\) VALUES \(\$1, \$2\)`).
		ExpectExec().WithArgs(sqlmock.AnyArg(), nomID1).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	dep, err := repo.Create("Новый Отдел", []string{nomID1.String()})
	require.NoError(t, err)
	require.NotNil(t, dep)
	assert.Equal(t, "Новый Отдел", dep.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDepartmentRepository_Update(t *testing.T) {
	// Обновление наименования подразделения и его номенклатуры
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(&database.DB{DB: db})
	depID := uuid.New()
	nomID1 := uuid.New()
	now := time.Now()

	mock.ExpectBegin()

	mock.ExpectQuery(`UPDATE departments SET name = \$2, updated_at = NOW\(\) WHERE id = \$1 RETURNING id, name, created_at, updated_at`).
		WithArgs(depID, "Обновленный Отдел").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).AddRow(depID, "Обновленный Отдел", now, now))

	mock.ExpectExec(`DELETE FROM department_nomenclature WHERE department_id = \$1`).WithArgs(depID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectPrepare(`INSERT INTO department_nomenclature \(department_id, nomenclature_id\) VALUES \(\$1, \$2\)`).
		ExpectExec().WithArgs(depID, nomID1).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	dep, err := repo.Update(depID, "Обновленный Отдел", []string{nomID1.String()})
	require.NoError(t, err)
	require.NotNil(t, dep)
	assert.Equal(t, "Обновленный Отдел", dep.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDepartmentRepository_Delete(t *testing.T) {
	// Удаление подразделения по его ID
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(&database.DB{DB: db})
	depID := uuid.New()

	mock.ExpectExec(`DELETE FROM departments WHERE id = \$1`).WithArgs(depID).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(depID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDepartmentRepository_Errors(t *testing.T) {
	// Проверка обработки ошибок при работе с подразделениями
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(&database.DB{DB: db})
	depID := uuid.New()

	t.Run("GetAll error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrConnDone)
		res, err := repo.GetAll()
		require.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("Create invalid nomenclature id", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO departments`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).AddRow(uuid.New(), "IT", time.Now(), time.Now()))
		
		mock.ExpectPrepare(`INSERT INTO department_nomenclature`)
		
		res, err := repo.Create("IT", []string{"invalid-uuid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid nomenclature id")
		assert.Nil(t, res)
	})

	t.Run("Update not found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`UPDATE departments`).WillReturnError(sql.ErrNoRows)
		
		res, err := repo.Update(depID, "IT", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "department not found")
		assert.Nil(t, res)
	})

	t.Run("Delete not found", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM departments`).WillReturnResult(sqlmock.NewResult(0, 0))
		
		err = repo.Delete(depID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "department not found")
	})
}

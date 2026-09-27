package repository

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepartmentRepository_GetAll(t *testing.T) {
	// Связи для нескольких подразделений загружаются одним дополнительным запросом.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(database.Wrap(db))
	firstID, secondID := uuid.New(), uuid.New()
	firstNomID, secondNomID := uuid.New(), uuid.New()

	mock.ExpectQuery(`SELECT id, name FROM departments ORDER BY name ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(firstID, "IT Отдел").AddRow(secondID, "Юридический отдел"))

	linksQuery := `SELECT dn.department_id, dn.nomenclature_id
		FROM department_nomenclature dn
		JOIN nomenclature n ON n.id = dn.nomenclature_id
		WHERE dn.department_id = ANY($1)
		ORDER BY dn.department_id, n.index`
	mock.ExpectQuery(regexp.QuoteMeta(linksQuery)).WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"department_id", "nomenclature_id"}).
			AddRow(firstID, firstNomID).AddRow(firstID, secondNomID))

	departments, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, departments, 2)
	assert.Equal(t, "IT Отдел", departments[0].Name)
	assert.Equal(t, []string{firstNomID.String(), secondNomID.String()}, departments[0].NomenclatureIDs)
	assert.Empty(t, departments[1].NomenclatureIDs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDepartmentRepository_GetNomenclatureIDs(t *testing.T) {
	// Получение массива идентификаторов номенклатур для подразделения
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(database.Wrap(db))
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

	repo := NewDepartmentRepository(database.Wrap(db))
	nomID1 := uuid.New()

	mock.ExpectBegin()

	mock.ExpectQuery(`INSERT INTO departments \(id, name\) VALUES \(\$1, \$2\) RETURNING id, name`).
		WithArgs(sqlmock.AnyArg(), "Новый Отдел").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uuid.New(), "Новый Отдел"))

	mock.ExpectPrepare(`INSERT INTO department_nomenclature \(department_id, nomenclature_id\) VALUES \(\$1, \$2\)`).
		ExpectExec().WithArgs(sqlmock.AnyArg(), nomID1).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	dep, err := repo.CreateWithOutbox("Новый Отдел", []string{nomID1.String()}, nil)
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

	repo := NewDepartmentRepository(database.Wrap(db))
	depID := uuid.New()
	nomID1 := uuid.New()

	mock.ExpectBegin()

	mock.ExpectQuery(`UPDATE departments SET name = \$2 WHERE id = \$1 RETURNING id, name`).
		WithArgs(depID, "Обновленный Отдел").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(depID, "Обновленный Отдел"))

	mock.ExpectExec(`DELETE FROM department_nomenclature WHERE department_id = \$1`).WithArgs(depID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectPrepare(`INSERT INTO department_nomenclature \(department_id, nomenclature_id\) VALUES \(\$1, \$2\)`).
		ExpectExec().WithArgs(depID, nomID1).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	dep, err := repo.UpdateWithOutbox(depID, "Обновленный Отдел", []string{nomID1.String()}, nil)
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

	repo := NewDepartmentRepository(database.Wrap(db))
	depID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM departments WHERE id = \$1`).WithArgs(depID).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err = repo.DeleteWithOutbox(depID, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDepartmentRepository_Errors(t *testing.T) {
	// Проверка обработки ошибок при работе с подразделениями
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDepartmentRepository(database.Wrap(db))
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
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uuid.New(), "IT"))

		mock.ExpectPrepare(`INSERT INTO department_nomenclature`)

		res, err := repo.CreateWithOutbox("IT", []string{"invalid-uuid"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid nomenclature id")
		assert.Nil(t, res)
	})

	t.Run("Update not found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`UPDATE departments`).WillReturnError(sql.ErrNoRows)

		res, err := repo.UpdateWithOutbox(depID, "IT", nil, nil)
		require.Error(t, err)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", appErr.Kind)
		assert.Equal(t, 404, appErr.Code)
		assert.Nil(t, res)
	})

	t.Run("Delete not found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM departments`).WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectRollback()

		err = repo.DeleteWithOutbox(depID, nil)
		require.Error(t, err)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", appErr.Kind)
		assert.Equal(t, 404, appErr.Code)
	})
}

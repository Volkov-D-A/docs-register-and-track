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

func TestNomenclatureRepository_GetAll(t *testing.T) {
	// Получение списка всех дел номенклатуры (с опциональной фильтрацией по году и направлению)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(&database.DB{DB: db})
	now := time.Now()

	t.Run("without filters", func(t *testing.T) {
		query := `SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature WHERE 1=1 ORDER BY index`

		rows := sqlmock.NewRows([]string{
			"id", "name", "index", "year", "direction", "next_number", "is_active", "created_at", "updated_at",
		}).AddRow(uuid.New(), "Офис", "01-01", 2024, "incoming", 1, true, now, now)

		mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

		res, err := repo.GetAll(0, "")
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with filters", func(t *testing.T) {
		query := `SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature WHERE 1=1 AND year = \$1 AND direction = \$2 ORDER BY index`

		rows := sqlmock.NewRows([]string{
			"id", "name", "index", "year", "direction", "next_number", "is_active", "created_at", "updated_at",
		}).AddRow(uuid.New(), "Офис", "01-01", 2024, "incoming", 1, true, now, now)

		mock.ExpectQuery(query).WithArgs(2024, "incoming").WillReturnRows(rows)

		res, err := repo.GetAll(2024, "incoming")
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNomenclatureRepository_GetByID(t *testing.T) {
	// Получение дела номенклатуры по его ID
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(&database.DB{DB: db})
	id := uuid.New()
	now := time.Now()

	query := `SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature WHERE id = \$1`

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "name", "index", "year", "direction", "next_number", "is_active", "created_at", "updated_at",
		}).AddRow(id, "Офис", "01-01", 2024, "incoming", 1, true, now, now)

		mock.ExpectQuery(query).WithArgs(id).WillReturnRows(rows)

		item, err := repo.GetByID(id)
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, id, item.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(query).WithArgs(id).WillReturnError(sql.ErrNoRows)

		item, err := repo.GetByID(id)
		require.NoError(t, err)
		require.Nil(t, item)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNomenclatureRepository_Create(t *testing.T) {
	// Создание нового дела номенклатуры
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(&database.DB{DB: db})
	id := uuid.New()
	now := time.Now()

	createQuery := `INSERT INTO nomenclature \(name, index, year, direction\)
		VALUES \(\$1, \$2, \$3, \$4\)
		RETURNING id`
	mock.ExpectQuery(createQuery).WithArgs("Тест", "02-12", 2025, "outgoing").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

	getQuery := `SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature WHERE id = \$1`
	mock.ExpectQuery(getQuery).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "index", "year", "direction", "next_number", "is_active", "created_at", "updated_at"}).
			AddRow(id, "Тест", "02-12", 2025, "outgoing", 1, true, now, now),
	)

	item, err := repo.Create("Тест", "02-12", 2025, "outgoing")
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, id, item.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNomenclatureRepository_Update(t *testing.T) {
	// Обновление параметров существующего дела номенклатуры
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(&database.DB{DB: db})
	id := uuid.New()
	now := time.Now()

	updateQuery := `UPDATE nomenclature SET name = \$1, index = \$2, year = \$3, direction = \$4, is_active = \$5, updated_at = \$6
		WHERE id = \$7`
	mock.ExpectExec(updateQuery).WithArgs("Обновлено", "02-12", 2025, "outgoing", false, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	getQuery := `SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature WHERE id = \$1`
	mock.ExpectQuery(getQuery).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "index", "year", "direction", "next_number", "is_active", "created_at", "updated_at"}).
			AddRow(id, "Обновлено", "02-12", 2025, "outgoing", 1, false, now, now),
	)

	item, err := repo.Update(id, "Обновлено", "02-12", 2025, "outgoing", false)
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, id, item.ID)
	assert.False(t, item.IsActive)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNomenclatureRepository_Delete(t *testing.T) {
	// Удаление дела номенклатуры
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(&database.DB{DB: db})
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM nomenclature WHERE id = \$1`).WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(id)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNomenclatureRepository_GetNextNumber(t *testing.T) {
	// Атомарное получение следующего порядкового номера для регистрации по делу
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(&database.DB{DB: db})
	id := uuid.New()

	query := `UPDATE nomenclature SET next_number = next_number \+ 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = \$1
		RETURNING next_number - 1, index`

	mock.ExpectQuery(query).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"next_number", "index"}).AddRow(5, "01-01"),
	)

	num, idx, err := repo.GetNextNumber(id)
	require.NoError(t, err)
	assert.Equal(t, 5, num)
	assert.Equal(t, "01-01", idx)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNomenclatureRepository_GetActiveByDirection(t *testing.T) {
	// Получение активных дел номенклатуры для указанного направления и года
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(&database.DB{DB: db})
	now := time.Now()

	query := `SELECT id, name, index, year, direction, next_number, is_active, created_at, updated_at
		FROM nomenclature
		WHERE direction = \$1 AND year = \$2 AND is_active = true
		ORDER BY index`

	rows := sqlmock.NewRows([]string{
		"id", "name", "index", "year", "direction", "next_number", "is_active", "created_at", "updated_at",
	}).AddRow(uuid.New(), "Офис", "01-01", 2024, "incoming", 1, true, now, now)

	mock.ExpectQuery(query).WithArgs("incoming", 2024).WillReturnRows(rows)

	res, err := repo.GetActiveByDirection("incoming", 2024)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

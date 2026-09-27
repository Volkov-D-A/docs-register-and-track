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

func TestNomenclatureRepository_GetAll(t *testing.T) {
	// Получение списка всех дел номенклатуры (с опциональной фильтрацией по виду документа)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(database.Wrap(db))

	t.Run("without filters", func(t *testing.T) {
		query := `SELECT id, name, index, year, kind_code, separator, numbering_mode, next_number, is_active
		FROM nomenclature WHERE 1=1 ORDER BY index`

		rows := sqlmock.NewRows([]string{
			"id", "name", "index", "year", "kind_code", "separator", "numbering_mode", "next_number", "is_active",
		}).AddRow(uuid.New(), "Офис", "01-01", 2024, "incoming_letter", "/", "index_and_number", 1, true)

		mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

		res, err := repo.GetAll(0, "")
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with filters", func(t *testing.T) {
		query := `SELECT id, name, index, year, kind_code, separator, numbering_mode, next_number, is_active
		FROM nomenclature WHERE 1=1 AND year = \$1 AND kind_code = \$2 ORDER BY index`

		rows := sqlmock.NewRows([]string{
			"id", "name", "index", "year", "kind_code", "separator", "numbering_mode", "next_number", "is_active",
		}).AddRow(uuid.New(), "Офис", "01-01", 2024, "incoming_letter", "/", "index_and_number", 1, true)

		mock.ExpectQuery(query).WithArgs(2024, "incoming_letter").WillReturnRows(rows)

		res, err := repo.GetAll(2024, "incoming_letter")
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

	repo := NewNomenclatureRepository(database.Wrap(db))
	id := uuid.New()

	query := `SELECT id, name, index, year, kind_code, separator, numbering_mode, next_number, is_active
		FROM nomenclature WHERE id = \$1`

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "name", "index", "year", "kind_code", "separator", "numbering_mode", "next_number", "is_active",
		}).AddRow(id, "Офис", "01-01", 2024, "incoming_letter", "/", "index_and_number", 1, true)

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

	repo := NewNomenclatureRepository(database.Wrap(db))
	id := uuid.New()

	createQuery := `INSERT INTO nomenclature \(name, index, year, kind_code, separator, numbering_mode, next_number\)
		VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)
		RETURNING id`
	mock.ExpectBegin()
	mock.ExpectQuery(createQuery).WithArgs("Тест", "02-12", 2025, "outgoing_letter", "-", "number_only", 7).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

	getQuery := `SELECT id, name, index, year, kind_code, separator, numbering_mode, next_number, is_active
		FROM nomenclature WHERE id = \$1`
	mock.ExpectCommit()
	mock.ExpectQuery(getQuery).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "index", "year", "kind_code", "separator", "numbering_mode", "next_number", "is_active"}).
			AddRow(id, "Тест", "02-12", 2025, "outgoing_letter", "-", "number_only", 7, true),
	)

	item, err := repo.CreateWithOutbox("Тест", "02-12", 2025, "outgoing_letter", "-", "number_only", 7, nil)
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, id, item.ID)
	assert.Equal(t, 7, item.NextNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNomenclatureRepositoryCreateWithOutboxRollsBackOnEnqueueFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewNomenclatureRepository(database.Wrap(db))
	repo.SetOutbox(NewOutboxRepository(database.Wrap(db)))
	event := models.OutboxEvent{EventType: models.OutboxEventAudit, DeduplicationKey: "nomenclature:test:create", Payload: `{}`}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO nomenclature`).WithArgs("Дело", "01-01", 2026, "incoming_letter", "/", "manual_only", 1).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs(event.EventType, event.DeduplicationKey, event.Payload).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err = repo.CreateWithOutbox("Дело", "01-01", 2026, "incoming_letter", "/", "manual_only", 1, []models.OutboxEvent{event})
	require.ErrorIs(t, err, assert.AnError)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNomenclatureRepository_Update(t *testing.T) {
	// Обновление параметров существующего дела номенклатуры
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(database.Wrap(db))
	id := uuid.New()

	updateQuery := `UPDATE nomenclature SET name = \$1, index = \$2, year = \$3, kind_code = \$4, separator = \$5, numbering_mode = \$6, is_active = \$7
		WHERE id = \$8`
	mock.ExpectBegin()
	mock.ExpectExec(updateQuery).WithArgs("Обновлено", "02-12", 2025, "outgoing_letter", "-", "number_only", false, id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	getQuery := `SELECT id, name, index, year, kind_code, separator, numbering_mode, next_number, is_active
		FROM nomenclature WHERE id = \$1`
	mock.ExpectCommit()
	mock.ExpectQuery(getQuery).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "index", "year", "kind_code", "separator", "numbering_mode", "next_number", "is_active"}).
			AddRow(id, "Обновлено", "02-12", 2025, "outgoing_letter", "-", "number_only", 1, false),
	)

	item, err := repo.UpdateWithOutbox(id, "Обновлено", "02-12", 2025, "outgoing_letter", "-", "number_only", false, nil)
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

	repo := NewNomenclatureRepository(database.Wrap(db))
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM nomenclature WHERE id = \$1`).WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err = repo.DeleteWithOutbox(id, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNomenclatureRepository_GetActiveByKind(t *testing.T) {
	// Получение активных дел номенклатуры для указанного вида документа и года
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewNomenclatureRepository(database.Wrap(db))

	query := `SELECT id, name, index, year, kind_code, separator, numbering_mode, next_number, is_active
		FROM nomenclature
		WHERE kind_code = \$1 AND year = \$2 AND is_active = true
		ORDER BY index`

	rows := sqlmock.NewRows([]string{
		"id", "name", "index", "year", "kind_code", "separator", "numbering_mode", "next_number", "is_active",
	}).AddRow(uuid.New(), "Офис", "01-01", 2024, "incoming_letter", "/", "index_and_number", 1, true)

	mock.ExpectQuery(query).WithArgs("incoming_letter", 2024).WillReturnRows(rows)

	res, err := repo.GetActiveByKind("incoming_letter", 2024)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

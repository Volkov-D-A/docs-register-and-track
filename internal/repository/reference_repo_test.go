package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === Document Types ===

func TestReferenceRepository_GetAllDocumentTypes(t *testing.T) {
	// Получение всех типов документов из кода.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})

	types, err := repo.GetAllDocumentTypes()
	require.NoError(t, err)
	require.Len(t, types, len(models.AllowedDocumentTypes()))
	assert.Equal(t, models.DocumentTypeLetter, types[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_CreateDocumentType(t *testing.T) {
	// Типы документов заданы в коде и не редактируются.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})

	item, err := repo.CreateDocumentType("Справка")
	require.Error(t, err)
	assert.Nil(t, item)
	assert.Contains(t, err.Error(), "не редактируются")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_UpdateDocumentType(t *testing.T) {
	// Типы документов заданы в коде и не редактируются.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	id := uuid.New()

	err = repo.UpdateDocumentType(id, "Новое Имя")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "не редактируются")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_DeleteDocumentType(t *testing.T) {
	// Типы документов заданы в коде и не редактируются.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	id := uuid.New()

	err = repo.DeleteDocumentType(id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "не редактируются")
	require.NoError(t, mock.ExpectationsWereMet())
}

// === Organizations ===

func TestReferenceRepository_GetAllOrganizations(t *testing.T) {
	// Получение полного списка организаций-корреспондентов
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	now := time.Now()

	query := `SELECT id, name, created_at FROM organizations ORDER BY name`
	rows := sqlmock.NewRows([]string{"id", "name", "created_at"}).
		AddRow(uuid.New(), "Организация А", now).
		AddRow(uuid.New(), "Организация Б", now)

	mock.ExpectQuery(query).WillReturnRows(rows)

	orgs, err := repo.GetAllOrganizations()
	require.NoError(t, err)
	require.Len(t, orgs, 2)
	assert.Equal(t, "Организация А", orgs[0].Name)
	assert.Equal(t, "Организация Б", orgs[1].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_FindOrCreateOrganization(t *testing.T) {
	// Поиск организации-корреспондента по названию или создание новой
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	id := uuid.New()
	name := "Google Ltd"
	now := time.Now()

	t.Run("found existing", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name, created_at FROM organizations WHERE name = \$1`).
			WithArgs(name).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at"}).AddRow(id, name, now))

		org, err := repo.FindOrCreateOrganization(name)
		require.NoError(t, err)
		require.NotNil(t, org)
		assert.Equal(t, id, org.ID)
		assert.Equal(t, name, org.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create new", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name, created_at FROM organizations WHERE name = \$1`).
			WithArgs(name).WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery(`INSERT INTO organizations \(name\) VALUES \(\$1\) RETURNING id`).
			WithArgs(name).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

		mock.ExpectQuery(`SELECT id, name, created_at FROM organizations WHERE id = \$1`).
			WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at"}).AddRow(id, name, now))

		org, err := repo.FindOrCreateOrganization(name)
		require.NoError(t, err)
		require.NotNil(t, org)
		assert.Equal(t, id, org.ID)
		assert.Equal(t, name, org.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReferenceRepository_SearchOrganizations(t *testing.T) {
	// Поиск организаций-корреспондентов по частичному совпадению (для подсказок)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	now := time.Now()
	query := `SELECT id, name, created_at FROM organizations WHERE name ILIKE \$1 ORDER BY name LIMIT 20`

	rows := sqlmock.NewRows([]string{"id", "name", "created_at"}).
		AddRow(uuid.New(), "Test Org", now)

	mock.ExpectQuery(query).WithArgs("%Test%").WillReturnRows(rows)

	orgs, err := repo.SearchOrganizations("Test")
	require.NoError(t, err)
	require.Len(t, orgs, 1)
	assert.Equal(t, "Test Org", orgs[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_UpdateOrganization(t *testing.T) {
	// Обновление карточки организации-корреспондента
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	id := uuid.New()

	mock.ExpectExec(`UPDATE organizations SET name = \$1 WHERE id = \$2`).
		WithArgs("Новое Название", id).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateOrganization(id, "Новое Название")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_DeleteOrganization(t *testing.T) {
	// Удаление организации-корреспондента из справочника
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
		WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.DeleteOrganization(id)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// === Resolution executors ===

func TestReferenceRepository_GetAllResolutionExecutors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	now := time.Now()

	mock.ExpectQuery(`SELECT id, name, created_at FROM resolution_executors ORDER BY name`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at"}).
			AddRow(uuid.New(), "Исполнитель А", now).
			AddRow(uuid.New(), "Исполнитель Б", now))

	items, err := repo.GetAllResolutionExecutors()

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "Исполнитель А", items[0].Name)
	assert.Equal(t, "Исполнитель Б", items[1].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_FindOrCreateResolutionExecutor(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	id := uuid.New()
	name := "Исполнитель"
	now := time.Now()

	t.Run("found existing", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name, created_at FROM resolution_executors WHERE name = \$1`).
			WithArgs(name).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at"}).AddRow(id, name, now))

		item, err := repo.FindOrCreateResolutionExecutor(name)

		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, id, item.ID)
		assert.Equal(t, name, item.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create new", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name, created_at FROM resolution_executors WHERE name = \$1`).
			WithArgs(name).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`INSERT INTO resolution_executors \(name\) VALUES \(\$1\) RETURNING id`).
			WithArgs(name).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
		mock.ExpectQuery(`SELECT id, name, created_at FROM resolution_executors WHERE id = \$1`).
			WithArgs(id).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at"}).AddRow(id, name, now))

		item, err := repo.FindOrCreateResolutionExecutor(name)

		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, id, item.ID)
		assert.Equal(t, name, item.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReferenceRepository_SearchResolutionExecutors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	now := time.Now()

	mock.ExpectQuery(`SELECT id, name, created_at FROM resolution_executors\s+WHERE name ILIKE \$1\s+ORDER BY name LIMIT 20`).
		WithArgs("%Исп%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at"}).
			AddRow(uuid.New(), "Исполнитель", now))

	items, err := repo.SearchResolutionExecutors("Исп")

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "Исполнитель", items[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_UpdateResolutionExecutor(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	id := uuid.New()

	mock.ExpectExec(`UPDATE resolution_executors SET name = \$1 WHERE id = \$2`).
		WithArgs("Новое имя", id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateResolutionExecutor(id, "Новое имя")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceRepository_DeleteResolutionExecutor(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(&database.DB{DB: db})
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM resolution_executors WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.DeleteResolutionExecutor(id)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

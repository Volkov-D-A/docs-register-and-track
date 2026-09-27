package repository

import (
	"database/sql"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === Organizations ===

func TestReferenceRepository_GetAllOrganizations(t *testing.T) {
	// Получение полного списка организаций-корреспондентов
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(database.Wrap(db))

	query := `SELECT id, name FROM organizations ORDER BY name`
	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(uuid.New(), "Организация А").
		AddRow(uuid.New(), "Организация Б")

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

	repo := NewReferenceRepository(database.Wrap(db))
	id := uuid.New()
	name := "Google Ltd"

	t.Run("found existing", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name FROM organizations WHERE name = \$1`).
			WithArgs(name).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, name))

		org, err := repo.FindOrCreateOrganization(name)
		require.NoError(t, err)
		require.NotNil(t, org)
		assert.Equal(t, id, org.ID)
		assert.Equal(t, name, org.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create new", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name FROM organizations WHERE name = \$1`).
			WithArgs(name).WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery(`INSERT INTO organizations \(name\) VALUES \(\$1\) RETURNING id`).
			WithArgs(name).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

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

	repo := NewReferenceRepository(database.Wrap(db))
	query := `SELECT id, name FROM organizations WHERE name ILIKE \$1 ORDER BY name LIMIT 20`

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(uuid.New(), "Test Org")

	mock.ExpectQuery(query).WithArgs("%Test%").WillReturnRows(rows)

	orgs, err := repo.SearchOrganizations("Test")
	require.NoError(t, err)
	require.Len(t, orgs, 1)
	assert.Equal(t, "Test Org", orgs[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

// === Resolution executors ===

func TestReferenceRepository_GetAllResolutionExecutors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewReferenceRepository(database.Wrap(db))

	mock.ExpectQuery(`SELECT id, name FROM resolution_executors ORDER BY name`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(uuid.New(), "Исполнитель А").
			AddRow(uuid.New(), "Исполнитель Б"))

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

	repo := NewReferenceRepository(database.Wrap(db))
	id := uuid.New()
	name := "Исполнитель"

	t.Run("found existing", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name FROM resolution_executors WHERE name = \$1`).
			WithArgs(name).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, name))

		item, err := repo.FindOrCreateResolutionExecutor(name)

		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, id, item.ID)
		assert.Equal(t, name, item.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create new", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name FROM resolution_executors WHERE name = \$1`).
			WithArgs(name).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`INSERT INTO resolution_executors \(name\) VALUES \(\$1\) RETURNING id`).
			WithArgs(name).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

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

	repo := NewReferenceRepository(database.Wrap(db))

	mock.ExpectQuery(`SELECT id, name FROM resolution_executors\s+WHERE name ILIKE \$1\s+ORDER BY name LIMIT 20`).
		WithArgs("%Исп%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(uuid.New(), "Исполнитель"))

	items, err := repo.SearchResolutionExecutors("Исп")

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "Исполнитель", items[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

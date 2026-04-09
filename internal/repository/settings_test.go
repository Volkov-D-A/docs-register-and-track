package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsRepository_Get(t *testing.T) {
	// Получение значения системной настройки по её ключу
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSettingsRepository(&database.DB{DB: db})
	now := time.Now()

	query := `SELECT key, value, description, updated_at FROM system_settings WHERE key = \$1`

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"key", "value", "description", "updated_at"}).
			AddRow("test_key", "test_val", "desc", now)

		mock.ExpectQuery(query).WithArgs("test_key").WillReturnRows(rows)

		s, err := repo.Get("test_key")
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, "test_key", s.Key)
		assert.Equal(t, "test_val", s.Value)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(query).WithArgs("test_key").WillReturnError(sql.ErrNoRows)

		s, err := repo.Get("test_key")
		require.Error(t, err)
		require.Nil(t, s)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSettingsRepository_GetAll(t *testing.T) {
	// Получение списка всех системных настроек
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSettingsRepository(&database.DB{DB: db})
	now := time.Now()

	query := `SELECT key, value, description, updated_at FROM system_settings ORDER BY key`

	rows := sqlmock.NewRows([]string{"key", "value", "description", "updated_at"}).
		AddRow("key1", "val1", "desc1", now).
		AddRow("key2", "val2", "desc2", now)

	mock.ExpectQuery(query).WillReturnRows(rows)

	settings, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, settings, 2)
	assert.Equal(t, "key1", settings[0].Key)
	assert.Equal(t, "key2", settings[1].Key)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettingsRepository_Update(t *testing.T) {
	// Обновление значения конкретной системной настройки
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSettingsRepository(&database.DB{DB: db})

	query := `INSERT INTO system_settings \(key, value, updated_at\)\s+VALUES \(\$1, \$2, NOW\(\)\)\s+ON CONFLICT \(key\) DO UPDATE\s+SET value = EXCLUDED.value, updated_at = NOW\(\)`
	mock.ExpectExec(query).WithArgs("test_key", "new_val").WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Update("test_key", "new_val")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

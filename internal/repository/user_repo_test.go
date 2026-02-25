package repository

import (
	"database/sql"
	"testing"
	"time"

	"docflow/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_GetByLogin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(&database.DB{DB: db})

	login := "testuser"
	id := uuid.New()
	now := time.Now()

	t.Run("success without department", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "login", "password_hash", "full_name", "is_active", "created_at", "updated_at",
			"d.id", "d.name",
		}).AddRow(
			id, login, "hash", "Test User", true, now, now,
			nil, nil, // нет подразделения
		)

		expectedQuery := `SELECT u.id, u.login, u.password_hash, u.full_name, u.is_active, u.created_at, u.updated_at,
	       d.id, d.name
	FROM users u
	LEFT JOIN departments d ON u.department_id = d.id WHERE u.login = \$1`

		mock.ExpectQuery(expectedQuery).WithArgs(login).WillReturnRows(rows)

		// Также вызывается GetUserRoles
		roleRows := sqlmock.NewRows([]string{"role"}).AddRow("admin")
		mock.ExpectQuery(`SELECT role FROM user_roles WHERE user_id = \$1`).WithArgs(id).WillReturnRows(roleRows)

		user, err := repo.GetByLogin(login)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, id, user.ID)
		assert.Equal(t, login, user.Login)
		assert.Equal(t, "Test User", user.FullName)
		assert.Equal(t, []string{"admin"}, user.Roles)
		assert.Nil(t, user.Department)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		expectedQuery := `SELECT u.id, u.login, u.password_hash, u.full_name, u.is_active, u.created_at, u.updated_at,
	       d.id, d.name
	FROM users u
	LEFT JOIN departments d ON u.department_id = d.id WHERE u.login = \$1`

		mock.ExpectQuery(expectedQuery).WithArgs("unknown").WillReturnError(sql.ErrNoRows)

		user, err := repo.GetByLogin("unknown")

		require.NoError(t, err)
		require.Nil(t, user)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

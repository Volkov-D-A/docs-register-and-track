package repository

import (
	"database/sql"
	"testing"
	"time"

	"docflow/internal/database"
	"docflow/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

func TestUserRepository_GetAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(&database.DB{DB: db})
	now := time.Now()
	uid := uuid.New()
	depID := uuid.New()

	mock.ExpectQuery(`SELECT(.*)FROM users u(.*)`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "login", "full_name", "is_active", "created_at", "updated_at",
			"d.id", "d.name",
		}).AddRow(uid, "user1", "User One", true, now, now, depID, "IT Dept"))

	// Expect roles
	mock.ExpectQuery(`SELECT(.*)FROM user_roles(.*)`).
		WithArgs(pq.Array([]uuid.UUID{uid})).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role"}).AddRow(uid, "admin"))

	// Expect department nomenclatures
	mock.ExpectQuery(`SELECT(.*)FROM department_nomenclature(.*)`).
		WithArgs(pq.Array([]uuid.UUID{depID})).
		WillReturnRows(sqlmock.NewRows([]string{"department_id", "nomenclature_id"}).AddRow(depID, uuid.New()))

	users, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "user1", users[0].Login)
	assert.Equal(t, []string{"admin"}, users[0].Roles)
	require.NotNil(t, users[0].Department)
	assert.Equal(t, "IT Dept", users[0].Department.Name)
	assert.Len(t, users[0].Department.NomenclatureIDs, 1)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(&database.DB{DB: db})
	
	req := models.CreateUserRequest{
		Login:    "newuser",
		Password: "Password123!",
		FullName: "New User",
		Roles:    []string{"user"},
	}

	mock.ExpectBegin()
	uid := uuid.New()

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(req.Login, sqlmock.AnyArg(), req.FullName, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uid))

	mock.ExpectExec(`INSERT INTO user_roles`).
		WithArgs(uid, "user").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	// getByCondition inside GetByID
	mock.ExpectQuery(`SELECT(.*)FROM users u(.*)`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "login", "password_hash", "full_name", "is_active", "created_at", "updated_at",
			"d.id", "d.name",
		}).AddRow(uid, req.Login, "hash", req.FullName, true, time.Now(), time.Now(), nil, nil))
	mock.ExpectQuery(`SELECT role FROM user_roles`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("user"))

	user, err := repo.Create(req)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, req.Login, user.Login)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(&database.DB{DB: db})
	uid := uuid.New()
	req := models.UpdateUserRequest{
		ID:       uid.String(),
		Login:    "upduser",
		FullName: "Upd User",
		IsActive: true,
		Roles:    []string{"admin"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE users SET`).
		WithArgs(req.Login, req.FullName, req.IsActive, nil, uid).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`DELETE FROM user_roles`).
		WithArgs(uid).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO user_roles`).
		WithArgs(uid, "admin").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	// getByCondition
	mock.ExpectQuery(`SELECT(.*)FROM users u(.*)`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "login", "password_hash", "full_name", "is_active", "created_at", "updated_at",
			"d.id", "d.name",
		}).AddRow(uid, req.Login, "hash", req.FullName, true, time.Now(), time.Now(), nil, nil))
	mock.ExpectQuery(`SELECT role FROM user_roles`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("admin"))

	user, err := repo.Update(req)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, req.Login, user.Login)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_OtherMethods(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(&database.DB{DB: db})
	uid := uuid.New()

	t.Run("UpdatePassword", func(t *testing.T) {
		mock.ExpectExec(`UPDATE users SET password_hash`).
			WithArgs("newhash", uid).
			WillReturnResult(sqlmock.NewResult(1, 1))
		
		err = repo.UpdatePassword(uid, "newhash")
		require.NoError(t, err)
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		mock.ExpectExec(`UPDATE users SET login(.*)full_name(.*)`).
			WithArgs("newlog", "newname", uid).
			WillReturnResult(sqlmock.NewResult(1, 1))
		
		err = repo.UpdateProfile(uid, models.UpdateProfileRequest{Login: "newlog", FullName: "newname"})
		require.NoError(t, err)
	})

	t.Run("CountUsers", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
		
		count, err := repo.CountUsers()
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})
	
	require.NoError(t, mock.ExpectationsWereMet())
}

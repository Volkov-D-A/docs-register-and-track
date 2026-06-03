package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func setupAdminAuditLogRepository(t *testing.T) (*AdminAuditLogRepository, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewAdminAuditLogRepository(&database.DB{DB: db}), mock, func() { db.Close() }
}

func TestAdminAuditLogRepository_Create(t *testing.T) {
	repo, mock, cleanup := setupAdminAuditLogRepository(t)
	defer cleanup()

	req := models.CreateAdminAuditLogRequest{
		UserID:   uuid.New(),
		UserName: "Администратор",
		Action:   "UPDATE_USER",
		Details:  "Изменен пользователь",
	}
	expectedID := uuid.New()

	mock.ExpectQuery(`INSERT INTO admin_audit_log \(user_id, user_name, action, details\)`).
		WithArgs(req.UserID, req.UserName, req.Action, req.Details).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedID))

	id, err := repo.Create(req)

	require.NoError(t, err)
	assert.Equal(t, expectedID, id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminAuditLogRepository_GetAll(t *testing.T) {
	repo, mock, cleanup := setupAdminAuditLogRepository(t)
	defer cleanup()

	t.Run("success", func(t *testing.T) {
		entryID := uuid.New()
		userID := uuid.New()
		now := time.Now()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM admin_audit_log`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(11))
		mock.ExpectQuery(`SELECT id, user_id, user_name, action, COALESCE\(details, ''\), created_at\s+FROM admin_audit_log`).
			WithArgs(10, 20).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "user_id", "user_name", "action", "details", "created_at",
			}).AddRow(entryID, userID, "Администратор", "UPDATE_USER", "details", now))

		entries, total, err := repo.GetAll(10, 20)

		require.NoError(t, err)
		assert.Equal(t, 11, total)
		require.Len(t, entries, 1)
		assert.Equal(t, entryID, entries[0].ID)
		assert.Equal(t, userID, entries[0].UserID)
		assert.Equal(t, "UPDATE_USER", entries[0].Action)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM admin_audit_log`).
			WillReturnError(sql.ErrConnDone)

		entries, total, err := repo.GetAll(10, 0)

		require.Error(t, err)
		assert.Nil(t, entries)
		assert.Zero(t, total)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

package repository

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func setupDocumentAccessRepository(t *testing.T) (*DocumentAccessRepository, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewDocumentAccessRepository(&database.DB{DB: db})
	return repo, mock, func() { db.Close() }
}

func TestDocumentAccessRepository_HasPermission(t *testing.T) {
	repo, mock, cleanup := setupDocumentAccessRepository(t)
	defer cleanup()

	t.Run("returns allowed flag", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(
				string(models.DocumentKindIncomingLetter),
				string(models.DocumentActionRead),
				"department-1",
				"user-1",
			).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		allowed, err := repo.HasPermission(
			string(models.DocumentKindIncomingLetter),
			string(models.DocumentActionRead),
			"department-1",
			"user-1",
		)

		require.NoError(t, err)
		assert.True(t, allowed)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps database error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(
				string(models.DocumentKindIncomingLetter),
				string(models.DocumentActionRead),
				"",
				"user-1",
			).
			WillReturnError(sql.ErrConnDone)

		allowed, err := repo.HasPermission(
			string(models.DocumentKindIncomingLetter),
			string(models.DocumentActionRead),
			"",
			"user-1",
		)

		require.Error(t, err)
		assert.False(t, allowed)
		assert.Contains(t, err.Error(), "failed to check document permission")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDocumentAccessRepository_HasSystemPermission(t *testing.T) {
	repo, mock, cleanup := setupDocumentAccessRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("user-1", models.SystemPermissionAdmin).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	allowed, err := repo.HasSystemPermission(models.SystemPermissionAdmin, "user-1")

	require.NoError(t, err)
	assert.True(t, allowed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentAccessRepository_GetUserAccessProfile(t *testing.T) {
	repo, mock, cleanup := setupDocumentAccessRepository(t)
	defer cleanup()

	t.Run("loads system and document rules", func(t *testing.T) {
		mock.ExpectQuery(`SELECT permission, is_allowed\s+FROM user_system_permissions`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows([]string{"permission", "is_allowed"}).
				AddRow(models.SystemPermissionAdmin, true).
				AddRow(models.SystemPermissionReferences, false))
		mock.ExpectQuery(`SELECT kind_code, action, is_allowed\s+FROM document_permissions`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows([]string{"kind_code", "action", "is_allowed"}).
				AddRow(string(models.DocumentKindIncomingLetter), string(models.DocumentActionRead), true).
				AddRow(string(models.DocumentKindOutgoingLetter), string(models.DocumentActionUpdate), false))

		profile, err := repo.GetUserAccessProfile("user-1")

		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, []models.UserSystemPermissionRule{
			{Permission: models.SystemPermissionAdmin, IsAllowed: true},
			{Permission: models.SystemPermissionReferences, IsAllowed: false},
		}, profile.SystemPermissions)
		assert.Equal(t, []models.UserDocumentPermissionRule{
			{KindCode: string(models.DocumentKindIncomingLetter), Action: string(models.DocumentActionRead), IsAllowed: true},
			{KindCode: string(models.DocumentKindOutgoingLetter), Action: string(models.DocumentActionUpdate), IsAllowed: false},
		}, profile.Permissions)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps system query error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT permission, is_allowed\s+FROM user_system_permissions`).
			WithArgs("user-1").
			WillReturnError(sql.ErrConnDone)

		profile, err := repo.GetUserAccessProfile("user-1")

		require.Error(t, err)
		assert.Nil(t, profile)
		assert.Contains(t, err.Error(), "failed to get user system permissions")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps document query error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT permission, is_allowed\s+FROM user_system_permissions`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows([]string{"permission", "is_allowed"}))
		mock.ExpectQuery(`SELECT kind_code, action, is_allowed\s+FROM document_permissions`).
			WithArgs("user-1").
			WillReturnError(sql.ErrConnDone)

		profile, err := repo.GetUserAccessProfile("user-1")

		require.Error(t, err)
		assert.Nil(t, profile)
		assert.Contains(t, err.Error(), "failed to get user document permissions")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDocumentAccessRepository_ReplaceUserAccessProfile(t *testing.T) {
	repo, mock, cleanup := setupDocumentAccessRepository(t)
	defer cleanup()

	t.Run("replaces rules in transaction", func(t *testing.T) {
		systemPermissions := []models.UserSystemPermissionRule{
			{Permission: models.SystemPermissionAdmin, IsAllowed: true},
			{Permission: models.SystemPermissionStatsDocuments, IsAllowed: false},
		}
		permissions := []models.UserDocumentPermissionRule{
			{KindCode: string(models.DocumentKindIncomingLetter), Action: string(models.DocumentActionRead), IsAllowed: true},
			{KindCode: string(models.DocumentKindOutgoingLetter), Action: string(models.DocumentActionUpdate), IsAllowed: false},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM user_system_permissions WHERE user_id = \$1`).
			WithArgs("user-1").
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO user_system_permissions`).
			WithArgs("user-1", models.SystemPermissionAdmin, true).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO user_system_permissions`).
			WithArgs("user-1", models.SystemPermissionStatsDocuments, false).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM document_permissions WHERE subject_type = 'user' AND subject_key = \$1`).
			WithArgs("user-1").
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO document_permissions`).
			WithArgs(string(models.DocumentKindIncomingLetter), "user-1", string(models.DocumentActionRead), true).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO document_permissions`).
			WithArgs(string(models.DocumentKindOutgoingLetter), "user-1", string(models.DocumentActionUpdate), false).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.ReplaceUserAccessProfile("user-1", systemPermissions, permissions)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back on insert error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM user_system_permissions WHERE user_id = \$1`).
			WithArgs("user-1").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(`INSERT INTO user_system_permissions`).
			WithArgs("user-1", models.SystemPermissionAdmin, true).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := repo.ReplaceUserAccessProfile("user-1", []models.UserSystemPermissionRule{
			{Permission: models.SystemPermissionAdmin, IsAllowed: true},
		}, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to insert user system permission")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

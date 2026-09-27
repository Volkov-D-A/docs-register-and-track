package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

func TestUserSubstitutionRepository_GetByPrincipalID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserSubstitutionRepository(database.Wrap(db))
	principalID := uuid.New()
	substituteID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT us\.substitute_user_id, us\.starts_at, us\.ends_at, us\.is_active`).
		WithArgs(principalID).
		WillReturnRows(sqlmock.NewRows([]string{
			"substitute_user_id", "starts_at", "ends_at", "is_active",
		}).AddRow(substituteID, now, nil, true))

	result, err := repo.GetByPrincipalID(principalID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, substituteID, result.SubstituteUserID)
	require.NotNil(t, result.StartsAt)
	assert.Nil(t, result.EndsAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserSubstitutionRepository_GetActivePrincipalIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserSubstitutionRepository(database.Wrap(db))
	substituteID := uuid.New()
	principalID := uuid.New()

	mock.ExpectQuery(`SELECT principal_user_id\s+FROM user_substitutions`).
		WithArgs(substituteID).
		WillReturnRows(sqlmock.NewRows([]string{"principal_user_id"}).AddRow(principalID))

	result, err := repo.GetActivePrincipalIDs(substituteID)

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{principalID}, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserSubstitutionRepository_ReplaceForPrincipalSQL(t *testing.T) {
	t.Run("deletes substitution when substitute is empty", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewUserSubstitutionRepository(database.Wrap(db))
		principalID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM user_substitutions WHERE principal_user_id = \$1`).
			WithArgs(principalID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()
		result, err := repo.ReplaceForPrincipalWithOutbox(principalID, nil, nil, nil, false, nil)

		require.NoError(t, err)
		assert.Nil(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("upserts substitution and reloads it", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewUserSubstitutionRepository(database.Wrap(db))
		principalID := uuid.New()
		substituteID := uuid.New()
		startsAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO user_substitutions`).
			WithArgs(principalID, substituteID, &startsAt, (*time.Time)(nil), true).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(`SELECT us\.substitute_user_id, us\.starts_at, us\.ends_at, us\.is_active`).
			WithArgs(principalID).
			WillReturnRows(sqlmock.NewRows([]string{
				"substitute_user_id", "starts_at", "ends_at", "is_active",
			}).AddRow(substituteID, startsAt, nil, true))

		result, err := repo.ReplaceForPrincipalWithOutbox(principalID, &substituteID, &startsAt, nil, true, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, substituteID, result.SubstituteUserID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserSubstitutionRepositoryReplaceForPrincipalWithOutboxRollsBackOnEnqueueFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserSubstitutionRepository(database.Wrap(db))
	repo.SetOutbox(NewOutboxRepository(repo.db))
	principalID, substituteID := uuid.New(), uuid.New()
	event := models.OutboxEvent{EventType: models.OutboxEventAudit, DeduplicationKey: "substitution:" + principalID.String(), Payload: `{"action":"update"}`}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO user_substitutions`).
		WithArgs(principalID, substituteID, (*time.Time)(nil), (*time.Time)(nil), true).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs(event.EventType, event.DeduplicationKey, event.Payload).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err = repo.ReplaceForPrincipalWithOutbox(principalID, &substituteID, nil, nil, true, []models.OutboxEvent{event})
	require.ErrorIs(t, err, assert.AnError)
	require.NoError(t, mock.ExpectationsWereMet())
}

package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestUserSubstitutionRepository_GetByPrincipalID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserSubstitutionRepository(&database.DB{DB: db})
	id := uuid.New()
	principalID := uuid.New()
	substituteID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT us\.id, us\.principal_user_id, us\.substitute_user_id,`).
		WithArgs(principalID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "principal_user_id", "substitute_user_id", "principal_name", "substitute_name",
			"starts_at", "ends_at", "is_active", "created_by", "created_at", "updated_at",
		}).AddRow(id, principalID, substituteID, "Principal", "Substitute", now, nil, true, nil, now, now))

	result, err := repo.GetByPrincipalID(principalID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, id, result.ID)
	assert.Equal(t, principalID, result.PrincipalUserID)
	assert.Equal(t, substituteID, result.SubstituteUserID)
	assert.Equal(t, "Principal", result.PrincipalName)
	assert.Equal(t, "Substitute", result.SubstituteName)
	require.NotNil(t, result.StartsAt)
	assert.Nil(t, result.EndsAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserSubstitutionRepository_GetActivePrincipalIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserSubstitutionRepository(&database.DB{DB: db})
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

func TestUserSubstitutionRepository_ReplaceForPrincipal(t *testing.T) {
	t.Run("deletes substitution when substitute is empty", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewUserSubstitutionRepository(&database.DB{DB: db})
		principalID := uuid.New()

		mock.ExpectExec(`DELETE FROM user_substitutions WHERE principal_user_id = \$1`).
			WithArgs(principalID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		result, err := repo.ReplaceForPrincipal(principalID, nil, nil, nil, false, nil)

		require.NoError(t, err)
		assert.Nil(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("upserts substitution and reloads it", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewUserSubstitutionRepository(&database.DB{DB: db})
		principalID := uuid.New()
		substituteID := uuid.New()
		createdBy := uuid.New()
		startsAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		now := time.Now()

		mock.ExpectExec(`INSERT INTO user_substitutions`).
			WithArgs(principalID, substituteID, &startsAt, (*time.Time)(nil), true, &createdBy).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(`SELECT us\.id, us\.principal_user_id, us\.substitute_user_id,`).
			WithArgs(principalID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "principal_user_id", "substitute_user_id", "principal_name", "substitute_name",
				"starts_at", "ends_at", "is_active", "created_by", "created_at", "updated_at",
			}).AddRow(uuid.New(), principalID, substituteID, "Principal", "Substitute", startsAt, nil, true, createdBy.String(), now, now))

		result, err := repo.ReplaceForPrincipal(principalID, &substituteID, &startsAt, nil, true, &createdBy)

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

	repo := NewUserSubstitutionRepository(&database.DB{DB: db})
	repo.SetOutbox(NewOutboxRepository(repo.db))
	principalID, substituteID := uuid.New(), uuid.New()
	event := models.OutboxEvent{EventType: models.OutboxEventAudit, DeduplicationKey: "substitution:" + principalID.String(), Payload: `{"action":"update"}`}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO user_substitutions`).
		WithArgs(principalID, substituteID, (*time.Time)(nil), (*time.Time)(nil), true, (*uuid.UUID)(nil)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs(event.EventType, event.DeduplicationKey, event.Payload).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err = repo.ReplaceForPrincipalWithOutbox(principalID, &substituteID, nil, nil, true, nil, []models.OutboxEvent{event})
	require.ErrorIs(t, err, assert.AnError)
	require.NoError(t, mock.ExpectationsWereMet())
}

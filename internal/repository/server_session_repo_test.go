package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
)

func TestServerSessionRepositoryLifecycle(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	repo := NewServerSessionRepository(&database.DB{DB: sqlDB})
	userID, sessionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	expiresAt := now.Add(12 * time.Hour)
	tokenHash := []byte("01234567890123456789012345678901")

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO server_sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id, created_at, last_seen_at")).
		WithArgs(userID, tokenHash, expiresAt).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "last_seen_at"}).AddRow(sessionID, now, now))
	session, err := repo.Create(userID, tokenHash, expiresAt)
	require.NoError(t, err)
	assert.Equal(t, sessionID, session.ID)

	mock.ExpectQuery("UPDATE server_sessions").WithArgs(tokenHash, now).WillReturnRows(
		sqlmock.NewRows([]string{"id", "user_id", "created_at", "expires_at", "revoked_at", "last_seen_at"}).AddRow(sessionID, userID, now, expiresAt, nil, now),
	)
	active, err := repo.GetActiveByTokenHash(tokenHash, now)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, userID, active.UserID)

	mock.ExpectExec("UPDATE server_sessions SET revoked_at").WithArgs(tokenHash, now).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.RevokeByTokenHash(tokenHash, now))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServerSessionRepositoryActivity(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	repo := NewServerSessionRepository(&database.DB{DB: sqlDB})
	now := time.Now().UTC()
	activeSince := now.Add(-15 * time.Minute)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\), COUNT\\(DISTINCT user_id\\)").
		WithArgs(now, activeSince).
		WillReturnRows(sqlmock.NewRows([]string{"sessions", "users"}).AddRow(5, 3))

	sessions, users, err := repo.Activity(now, activeSince)
	require.NoError(t, err)
	assert.Equal(t, 5, sessions)
	assert.Equal(t, 3, users)
	require.NoError(t, mock.ExpectationsWereMet())
}

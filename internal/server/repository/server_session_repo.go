package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

type ServerSessionRepository struct {
	db *database.DB
}

func NewServerSessionRepository(db *database.DB) *ServerSessionRepository {
	return &ServerSessionRepository{db: db}
}

func (r *ServerSessionRepository) Create(userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO server_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create server session: %w", err)
	}
	return nil
}

func (r *ServerSessionRepository) GetActiveByTokenHash(tokenHash []byte, now time.Time) (*models.ServerSession, error) {
	session := &models.ServerSession{}
	err := r.db.QueryRow(`
		UPDATE server_sessions
		SET last_seen_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2
		RETURNING user_id
	`, tokenHash, now).Scan(&session.UserID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active server session: %w", err)
	}
	return session, nil
}

func (r *ServerSessionRepository) RevokeByTokenHash(tokenHash []byte, now time.Time) error {
	_, err := r.db.Exec(`UPDATE server_sessions SET revoked_at = COALESCE(revoked_at, $2) WHERE token_hash = $1`, tokenHash, now)
	if err != nil {
		return fmt.Errorf("revoke server session: %w", err)
	}
	return nil
}

func (r *ServerSessionRepository) DeleteExpired(before time.Time) (int64, error) {
	result, err := r.db.Exec(`DELETE FROM server_sessions WHERE expires_at < $1 OR (revoked_at IS NOT NULL AND revoked_at < $1)`, before)
	if err != nil {
		return 0, fmt.Errorf("delete expired server sessions: %w", err)
	}
	return result.RowsAffected()
}

// Activity returns valid sessions and distinct recently active users.
func (r *ServerSessionRepository) Activity(now, activeSince time.Time) (int, int, error) {
	var sessions, users int
	err := r.db.QueryRow(`
		SELECT COUNT(*), COUNT(DISTINCT user_id) FILTER (WHERE last_seen_at >= $2)
		FROM server_sessions
		WHERE revoked_at IS NULL AND expires_at > $1
	`, now, activeSince).Scan(&sessions, &users)
	if err != nil {
		return 0, 0, fmt.Errorf("get server session activity: %w", err)
	}
	return sessions, users, nil
}

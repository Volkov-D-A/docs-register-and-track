package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type ServerSessionRepository struct {
	db *database.DB
}

func NewServerSessionRepository(db *database.DB) *ServerSessionRepository {
	return &ServerSessionRepository{db: db}
}

func (r *ServerSessionRepository) Create(userID uuid.UUID, tokenHash []byte, expiresAt time.Time) (*models.ServerSession, error) {
	session := &models.ServerSession{UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}
	err := r.db.QueryRow(`
		INSERT INTO server_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, last_seen_at
	`, userID, tokenHash, expiresAt).Scan(&session.ID, &session.CreatedAt, &session.LastSeenAt)
	if err != nil {
		return nil, fmt.Errorf("create server session: %w", err)
	}
	return session, nil
}

func (r *ServerSessionRepository) GetActiveByTokenHash(tokenHash []byte, now time.Time) (*models.ServerSession, error) {
	session := &models.ServerSession{TokenHash: tokenHash}
	err := r.db.QueryRow(`
		UPDATE server_sessions
		SET last_seen_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2
		RETURNING id, user_id, created_at, expires_at, revoked_at, last_seen_at
	`, tokenHash, now).Scan(&session.ID, &session.UserID, &session.CreatedAt, &session.ExpiresAt, &session.RevokedAt, &session.LastSeenAt)
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

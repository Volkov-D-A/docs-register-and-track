package models

import (
	"time"

	"github.com/google/uuid"
)

type ServerSession struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastSeenAt time.Time
}

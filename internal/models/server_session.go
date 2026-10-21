package models

import "github.com/google/uuid"

// ServerSession contains the principal of a session validated by the repository.
// Expiration, revocation and activity are checked and maintained in PostgreSQL.
type ServerSession struct {
	UserID uuid.UUID
}

package models

import (
	"time"

	"github.com/google/uuid"
)

// WorkspaceDocument is a bounded document preview under server-resolved read scopes.
type WorkspaceDocument struct {
	ID             uuid.UUID
	DocumentKind   string
	DocumentNumber string
	DocumentDate   time.Time
	RegisteredAt   time.Time
	Description    string
	Correspondents []string
}

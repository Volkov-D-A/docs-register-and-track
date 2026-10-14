package ports

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

// DocumentAccessPrincipal provides the request-local identity used by document
// access checks. The server supplies an immutable principal for each HTTP
// request; service tests provide a separate fake principal.
type DocumentAccessPrincipal interface {
	RequireAuthenticated() error
	GetCurrentUser() (*dto.User, error)
	GetCurrentUserUUID() (uuid.UUID, error)
}

// DocumentVisibilityStore resolves read from permissions, nomenclature,
// controls and the complete undirected graph of document links.
type DocumentVisibilityStore interface {
	CanReadDocument(userID, documentID uuid.UUID) (bool, error)
	GetReadableDocumentIDs(userID uuid.UUID, documentIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

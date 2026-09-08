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

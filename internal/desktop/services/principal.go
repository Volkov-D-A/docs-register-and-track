package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

// Principal is internal composition support and must never be registered with Wails.
// Schema readiness is owned by the server; migration controls remain available
// during maintenance.
type Principal struct {
	auth *AuthService
}

func NewPrincipal(auth *AuthService) *Principal { return &Principal{auth: auth} }

func (s *Principal) GetCurrentUser() (*dto.User, error) { return s.auth.GetCurrentUser() }

func (s *Principal) GetCurrentUserID() string { return s.auth.GetSessionState().UserID }

func (s *Principal) RequireSystemPermission(permission string) error {
	user, err := s.GetCurrentUser()
	if err != nil {
		return err
	}
	if user == nil || !user.IsActive {
		return models.ErrUnauthorized
	}
	id, err := uuid.Parse(user.ID)
	if err != nil || id == uuid.Nil {
		return models.ErrUnauthorized
	}
	for _, value := range user.SystemPermissions {
		if value == permission {
			return nil
		}
	}
	return models.ErrForbidden
}

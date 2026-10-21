package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

type documentUserReader interface {
	GetByID(uuid.UUID) (*models.User, error)
}

type documentPrincipalStub struct {
	currentUserID     uuid.UUID
	users             documentUserReader
	systemPermissions []string
}

func newDocumentPrincipalStub(users documentUserReader) *documentPrincipalStub {
	return &documentPrincipalStub{users: users}
}
func (p *documentPrincipalStub) GetCurrentUser() (*dto.User, error) {
	if p.currentUserID == uuid.Nil {
		return nil, models.ErrUnauthorized
	}
	user, err := p.users.GetByID(p.currentUserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, models.ErrUnauthorized
	}
	return dto.MapUser(user), nil
}
func (p *documentPrincipalStub) RequireAuthenticated() error {
	_, err := p.GetCurrentUser()
	return err
}
func (p *documentPrincipalStub) GetCurrentUserUUID() (uuid.UUID, error) {
	if err := p.RequireAuthenticated(); err != nil {
		return uuid.Nil, err
	}
	return p.currentUserID, nil
}
func (p *documentPrincipalStub) RequireSystemPermission(permission string) error {
	if err := p.RequireAuthenticated(); err != nil {
		return err
	}
	for _, allowed := range p.systemPermissions {
		if allowed == permission {
			return nil
		}
	}
	return models.ErrForbidden
}

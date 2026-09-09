package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
	"github.com/google/uuid"
)

type attachmentPrincipalStub struct {
	currentUserID uuid.UUID
	users         ports.UserStore
	access        ports.DocumentAccessStore
}

func newAttachmentPrincipalStub(users ports.UserStore) *attachmentPrincipalStub {
	return &attachmentPrincipalStub{users: users}
}
func (p *attachmentPrincipalStub) SetAccessStore(access ports.DocumentAccessStore) { p.access = access }
func (p *attachmentPrincipalStub) GetCurrentUser() (*dto.User, error) {
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
func (p *attachmentPrincipalStub) RequireAuthenticated() error {
	_, err := p.GetCurrentUser()
	return err
}
func (p *attachmentPrincipalStub) GetCurrentUserUUID() (uuid.UUID, error) {
	if err := p.RequireAuthenticated(); err != nil {
		return uuid.Nil, err
	}
	return p.currentUserID, nil
}
func (p *attachmentPrincipalStub) RequireSystemPermission(permission string) error {
	if err := p.RequireAuthenticated(); err != nil {
		return err
	}
	if p.access != nil {
		allowed, err := p.access.HasSystemPermission(permission, p.currentUserID.String())
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}
	}
	return models.ErrForbidden
}

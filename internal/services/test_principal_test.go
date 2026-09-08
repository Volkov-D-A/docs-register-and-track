package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

// testPrincipal supplies identity to business-service tests. It has no login,
// password verification, sessions, bootstrap, or database access.
type testPrincipal struct {
	currentUserID uuid.UUID
	userRepo      interface {
		GetByID(uuid.UUID) (*models.User, error)
	}
	accessRepo      DocumentAccessStore
	schemaLifecycle SchemaLifecycle
}

func newTestPrincipal(users interface {
	GetByID(uuid.UUID) (*models.User, error)
}) *testPrincipal {
	return &testPrincipal{userRepo: users}
}
func (p *testPrincipal) SetAccessStore(store DocumentAccessStore) { p.accessRepo = store }
func (p *testPrincipal) GetCurrentUser() (*dto.User, error) {
	if p.currentUserID == uuid.Nil {
		return nil, models.ErrUnauthorized
	}
	user, err := p.userRepo.GetByID(p.currentUserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		p.currentUserID = uuid.Nil
		return nil, models.ErrUnauthorized
	}
	return dto.MapUser(user), nil
}
func (p *testPrincipal) checkSchemaReady() error {
	if p.schemaLifecycle != nil {
		return p.schemaLifecycle.CheckReady()
	}
	return nil
}
func (p *testPrincipal) RequireAuthenticated() error {
	if err := p.checkSchemaReady(); err != nil {
		return err
	}
	_, err := p.GetCurrentUser()
	return err
}
func (p *testPrincipal) GetCurrentUserUUID() (uuid.UUID, error) {
	if err := p.RequireAuthenticated(); err != nil {
		return uuid.Nil, err
	}
	return p.currentUserID, nil
}
func (p *testPrincipal) GetCurrentUserID() string {
	if p.currentUserID == uuid.Nil {
		return ""
	}
	return p.currentUserID.String()
}
func (p *testPrincipal) GetCurrentAuditInfo() (uuid.UUID, string) {
	user, err := p.GetCurrentUser()
	if err != nil {
		return uuid.Nil, "system"
	}
	return p.currentUserID, user.FullName
}
func (p *testPrincipal) HasSystemPermissionFor(id uuid.UUID, permission string) bool {
	if p.accessRepo == nil {
		return false
	}
	allowed, err := p.accessRepo.HasSystemPermission(permission, id.String())
	return err == nil && allowed
}
func (p *testPrincipal) HasSystemPermission(permission string) bool {
	if _, err := p.GetCurrentUser(); err != nil {
		return false
	}
	return p.HasSystemPermissionFor(p.currentUserID, permission)
}
func (p *testPrincipal) RequireSystemPermissionWithoutSchemaCheck(permission string) error {
	if _, err := p.GetCurrentUser(); err != nil {
		return err
	}
	if !p.HasSystemPermissionFor(p.currentUserID, permission) {
		return models.ErrForbidden
	}
	return nil
}
func (p *testPrincipal) RequireSystemPermission(permission string) error {
	if err := p.checkSchemaReady(); err != nil {
		return err
	}
	return p.RequireSystemPermissionWithoutSchemaCheck(permission)
}
func (p *testPrincipal) HasAnySystemPermission(permissions ...string) bool {
	for _, v := range permissions {
		if p.HasSystemPermission(v) {
			return true
		}
	}
	return false
}
func (p *testPrincipal) RequireAnySystemPermission(permissions ...string) error {
	if err := p.RequireAuthenticated(); err != nil {
		return err
	}
	if !p.HasAnySystemPermission(permissions...) {
		return models.ErrForbidden
	}
	return nil
}
func (p *testPrincipal) Logout() error { p.currentUserID = uuid.Nil; return nil }

func (p *testPrincipal) IsAuthenticated() bool { return p.currentUserID != uuid.Nil }

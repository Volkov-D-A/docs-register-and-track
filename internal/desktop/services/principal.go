package services

import (
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

// Principal is internal composition support and must never be registered with Wails.
type Principal struct {
	auth            *AuthService
	schemaLifecycle interface{ CheckReady() error }
}

func NewPrincipal(auth *AuthService, readiness interface{ CheckReady() error }) *Principal {
	return &Principal{auth: auth, schemaLifecycle: readiness}
}
func (s *Principal) GetCurrentUser() (*dto.User, error) { return s.auth.GetCurrentUser() }

func (s *Principal) GetCurrentUserUUID() (uuid.UUID, error) {
	if err := s.checkSchemaReady(); err != nil {
		return uuid.Nil, err
	}
	principal, err := s.getActiveSessionPrincipal()
	if err != nil {
		return uuid.Nil, err
	}
	return principal.ID, nil
}

func (s *Principal) RequireAuthenticated() error {
	if err := s.checkSchemaReady(); err != nil {
		return err
	}
	_, err := s.getActiveSessionPrincipal()
	return err
}

func (s *Principal) checkSchemaReady() error {
	if s.schemaLifecycle == nil {
		return nil
	}
	return s.schemaLifecycle.CheckReady()
}

func (s *Principal) getActiveSessionPrincipal() (*models.SessionPrincipal, error) {
	if s.auth.serverAuth == nil {
		return nil, errServerAuthNotConfigured
	}

	ctx, cancel := s.auth.operationContext(15 * time.Second)
	defer cancel()
	user, err := s.auth.serverAuth.Me(ctx)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, models.ErrUnauthorized
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil || userID == uuid.Nil || !user.IsActive {
		return nil, models.ErrUnauthorized
	}
	return &models.SessionPrincipal{ID: userID, IsActive: true}, nil
}

func (s *Principal) GetCurrentUserID() string {
	return s.auth.GetSessionState().UserID
}

func (s *Principal) GetCurrentAuditInfo() (uuid.UUID, string) {
	if s.auth.serverAuth == nil {
		return uuid.Nil, "system"
	}

	ctx, cancel := s.auth.operationContext(15 * time.Second)
	defer cancel()
	user, err := s.auth.serverAuth.Me(ctx)
	if err != nil || user == nil {
		return uuid.Nil, "system"
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return uuid.Nil, "system"
	}
	return userID, user.FullName
}

func (s *Principal) HasSystemPermission(permission string) bool {
	principal, err := s.getActiveSessionPrincipal()
	if err != nil {
		return false
	}
	return s.HasSystemPermissionFor(principal.ID, permission)
}

func (s *Principal) HasSystemPermissionFor(userID uuid.UUID, permission string) bool {
	if s.auth.serverAuth == nil {
		return false
	}

	ctx, cancel := s.auth.operationContext(15 * time.Second)
	defer cancel()
	user, err := s.auth.serverAuth.Me(ctx)
	if err != nil || user == nil || user.ID != userID.String() {
		return false
	}
	for _, value := range user.SystemPermissions {
		if value == permission {
			return true
		}
	}
	return false
}

func (s *Principal) RequireSystemPermission(permission string) error {
	if err := s.checkSchemaReady(); err != nil {
		return err
	}
	return s.RequireSystemPermissionWithoutSchemaCheck(permission)
}

func (s *Principal) RequireSystemPermissionWithoutSchemaCheck(permission string) error {
	principal, err := s.getActiveSessionPrincipal()
	if err != nil {
		return err
	}
	if !s.HasSystemPermissionFor(principal.ID, permission) {
		return models.ErrForbidden
	}
	return nil
}

func (s *Principal) HasAnySystemPermission(permissions ...string) bool {
	principal, err := s.getActiveSessionPrincipal()
	if err != nil {
		return false
	}
	for _, permission := range permissions {
		if s.HasSystemPermissionFor(principal.ID, permission) {
			return true
		}
	}
	return false
}

func (s *Principal) RequireAnySystemPermission(permissions ...string) error {
	if err := s.checkSchemaReady(); err != nil {
		return err
	}
	principal, err := s.getActiveSessionPrincipal()
	if err != nil {
		return err
	}
	for _, permission := range permissions {
		if s.HasSystemPermissionFor(principal.ID, permission) {
			return nil
		}
	}
	return models.ErrForbidden
}

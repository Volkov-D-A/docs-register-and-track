package services

import (
	"fmt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentAccessAdminService управляет прямыми document-domain правами пользователей.
type DocumentAccessAdminService struct {
	auth       *AuthService
	accessRepo DocumentAccessStore
	userRepo   UserStore
}

// NewDocumentAccessAdminService создает новый сервис администрирования document access.
func NewDocumentAccessAdminService(auth *AuthService, accessRepo DocumentAccessStore, userRepo UserStore) *DocumentAccessAdminService {
	return &DocumentAccessAdminService{
		auth:       auth,
		accessRepo: accessRepo,
		userRepo:   userRepo,
	}
}

// GetUserAccessProfile возвращает прямые права пользователя в document-domain.
func (s *DocumentAccessAdminService) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	if _, err := parseUUID(userID); err != nil {
		return nil, err
	}

	return s.accessRepo.GetUserAccessProfile(userID)
}

// UpdateUserAccessProfile заменяет прямые document-domain права пользователя.
func (s *DocumentAccessAdminService) UpdateUserAccessProfile(req models.UpdateUserDocumentAccessRequest) error {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return err
	}

	uid, err := parseUUID(req.UserID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(uid)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("пользователь не найден")
	}

	return s.accessRepo.ReplaceUserAccessProfile(req.UserID, req.SystemPermissions, req.Permissions)
}

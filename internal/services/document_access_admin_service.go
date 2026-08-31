package services

import (
	"context"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// DocumentAccessAdminService управляет прямыми document-domain правами пользователей.
type DocumentAccessAdminService struct {
	auth       *AuthService
	accessRepo DocumentAccessStore
	userRepo   UserStore
	server     serverclient.UserAccessClient
}

func (s *DocumentAccessAdminService) SetServerClient(client serverclient.UserAccessClient) {
	s.server = client
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
	if s.server == nil {
		return nil, errServerUserAdministrationNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.GetUserAccessProfile(ctx, userID)
}

// UpdateUserAccessProfile заменяет прямые document-domain права пользователя.
func (s *DocumentAccessAdminService) UpdateUserAccessProfile(req models.UpdateUserDocumentAccessRequest) error {
	if s.server == nil {
		return errServerUserAdministrationNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.UpdateUserAccessProfile(ctx, req)
}

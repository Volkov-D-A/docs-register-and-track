package services

import (
	"context"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// UserSubstitutionService управляет замещающими исполнителями пользователя.
type UserSubstitutionService struct {
	server serverclient.UserSubstitutionAdminClient
}

func NewUserSubstitutionService(client serverclient.UserSubstitutionAdminClient) *UserSubstitutionService {
	return &UserSubstitutionService{server: client}
}

// GetMySubstitution возвращает настройку замещения текущего пользователя.
func (s *UserSubstitutionService) GetMySubstitution() (*dto.UserSubstitution, error) {
	if s.server == nil {
		return nil, errServerUserAdministrationNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.GetMySubstitution(ctx)
}

// UpdateMySubstitution обновляет замещающего текущего пользователя.
func (s *UserSubstitutionService) UpdateMySubstitution(req models.UpdateUserSubstitutionRequest) (*dto.UserSubstitution, error) {
	if s.server == nil {
		return nil, errServerUserAdministrationNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.UpdateMySubstitution(ctx, req)
}

// GetUserSubstitution возвращает настройку замещения выбранного пользователя. Доступно администратору.
func (s *UserSubstitutionService) GetUserSubstitution(userID string) (*dto.UserSubstitution, error) {
	if s.server == nil {
		return nil, errServerUserAdministrationNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.GetUserSubstitution(ctx, userID)
}

// UpdateUserSubstitution обновляет замещение выбранного пользователя. Доступно администратору.
func (s *UserSubstitutionService) UpdateUserSubstitution(req models.UpdateUserSubstitutionRequest) (*dto.UserSubstitution, error) {
	if s.server == nil {
		return nil, errServerUserAdministrationNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.UpdateUserSubstitution(ctx, req)
}

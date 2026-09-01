package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

var errServerUserAdministrationNotConfigured = errors.New("docflow-server user administration client is not configured")

// UserService предоставляет бизнес-логику для управления пользователями.
type UserService struct {
	userRepo  UserStore
	auth      *AuthService
	server    serverclient.UserClient
	executors serverclient.ExecutorClient
}

// NewUserService создает новый экземпляр UserService.
func NewUserService(userRepo UserStore, auth *AuthService) *UserService {
	return &UserService{
		userRepo: userRepo,
		auth:     auth,
	}
}

func (s *UserService) SetServerClient(client serverclient.UserClient) {
	s.server = client
	s.executors, _ = client.(serverclient.ExecutorClient)
}

func (s *UserService) serverClient() (serverclient.UserClient, error) {
	if s.server == nil {
		return nil, errServerUserAdministrationNotConfigured
	}
	return s.server, nil
}

// GetAllUsers возвращает список всех пользователей (доступно администраторам).
func (s *UserService) GetAllUsers() ([]dto.User, error) {
	client, err := s.serverClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return client.ListUsers(ctx)
}

// CreateUser создает нового пользователя (доступно только администраторам).
func (s *UserService) CreateUser(req models.CreateUserRequest) (*dto.User, error) {
	client, err := s.serverClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return client.CreateUser(ctx, req)
}

// UpdateUser обновляет данные пользователя (доступно только администраторам).
func (s *UserService) UpdateUser(req models.UpdateUserRequest) (*dto.User, error) {
	client, err := s.serverClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return client.UpdateUser(ctx, req)
}

// ResetPassword генерирует временный пароль пользователя (доступно только администраторам).
func (s *UserService) ResetPassword(userID string) (string, error) {
	client, err := s.serverClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return client.ResetUserPassword(ctx, userID)
}

// GetExecutors возвращает список активных сотрудников для назначений и ознакомления.
func (s *UserService) GetExecutors() ([]dto.User, error) {
	if s.executors == nil {
		return nil, errServerUserAdministrationNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.executors.ListExecutors(ctx)
}

// GetSubstitutionCandidates возвращает активных пользователей, которых можно выбрать замещающими.
func (s *UserService) GetSubstitutionCandidates() ([]dto.User, error) {
	client, err := s.serverClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return client.ListSubstitutionCandidates(ctx)
}

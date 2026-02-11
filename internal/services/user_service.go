package services

import (
	"context"

	"docflow/internal/models"
	"docflow/internal/repository"
)

type UserService struct {
	ctx      context.Context
	userRepo *repository.UserRepository
	auth     *AuthService
}

func NewUserService(userRepo *repository.UserRepository, auth *AuthService) *UserService {
	return &UserService{
		userRepo: userRepo,
		auth:     auth,
	}
}

// SetContext вызывается из OnStartup
func (s *UserService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// GetAllUsers — получить всех пользователей (только admin)
func (s *UserService) GetAllUsers() ([]models.User, error) {
	if !s.auth.HasRole("admin") {
		return nil, ErrNotAuthenticated
	}
	return s.userRepo.GetAll()
}

// CreateUser — создать пользователя (только admin)
func (s *UserService) CreateUser(req models.CreateUserRequest) (*models.User, error) {
	if !s.auth.HasRole("admin") {
		return nil, ErrNotAuthenticated
	}
	return s.userRepo.Create(req)
}

// UpdateUser — обновить пользователя (только admin)
func (s *UserService) UpdateUser(req models.UpdateUserRequest) (*models.User, error) {
	if !s.auth.HasRole("admin") {
		return nil, ErrNotAuthenticated
	}
	return s.userRepo.Update(req)
}

// ResetPassword — сброс пароля (только admin)
func (s *UserService) ResetPassword(userID string, newPassword string) error {
	if !s.auth.HasRole("admin") {
		return ErrNotAuthenticated
	}

	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}

	return s.userRepo.ResetPassword(uid, newPassword)
}

// GetExecutors — получить список исполнителей
func (s *UserService) GetExecutors() ([]models.User, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return s.userRepo.GetExecutors()
}

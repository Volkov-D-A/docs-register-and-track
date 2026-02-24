package services

import (
	"docflow/internal/dto"
	"docflow/internal/models"
	"docflow/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
	auth     *AuthService
}

func NewUserService(userRepo *repository.UserRepository, auth *AuthService) *UserService {
	return &UserService{
		userRepo: userRepo,
		auth:     auth,
	}
}

// GetAllUsers — получить всех пользователей (только admin или clerk)
func (s *UserService) GetAllUsers() ([]dto.User, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.GetAll()
	return dto.MapUsers(res), err
}

// CreateUser — создать пользователя (только admin)
func (s *UserService) CreateUser(req models.CreateUserRequest) (*dto.User, error) {
	if !s.auth.HasRole("admin") {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.Create(req)
	return dto.MapUser(res), err
}

// UpdateUser — обновить пользователя (только admin)
func (s *UserService) UpdateUser(req models.UpdateUserRequest) (*dto.User, error) {
	if !s.auth.HasRole("admin") {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.Update(req)
	return dto.MapUser(res), err
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
func (s *UserService) GetExecutors() ([]dto.User, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.GetExecutors()
	return dto.MapUsers(res), err
}

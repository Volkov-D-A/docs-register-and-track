package services

import (
	"docflow/internal/dto"
	"docflow/internal/models"
)

// UserService предоставляет бизнес-логику для управления пользователями.
type UserService struct {
	userRepo UserStore
	auth     *AuthService
}

// NewUserService создает новый экземпляр UserService.
func NewUserService(userRepo UserStore, auth *AuthService) *UserService {
	return &UserService{
		userRepo: userRepo,
		auth:     auth,
	}
}

// GetAllUsers возвращает список всех пользователей (доступно администраторам и делопроизводителям).
func (s *UserService) GetAllUsers() ([]dto.User, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.GetAll()
	return dto.MapUsers(res), err
}

// CreateUser создает нового пользователя (доступно только администраторам).
func (s *UserService) CreateUser(req models.CreateUserRequest) (*dto.User, error) {
	if !s.auth.HasRole("admin") {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.Create(req)
	return dto.MapUser(res), err
}

// UpdateUser обновляет данные пользователя (доступно только администраторам).
func (s *UserService) UpdateUser(req models.UpdateUserRequest) (*dto.User, error) {
	if !s.auth.HasRole("admin") {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.Update(req)
	return dto.MapUser(res), err
}

// ResetPassword сбрасывает пароль пользователя (доступно только администраторам).
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

// GetExecutors возвращает список пользователей-исполнителей.
func (s *UserService) GetExecutors() ([]dto.User, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.GetExecutors()
	return dto.MapUsers(res), err
}

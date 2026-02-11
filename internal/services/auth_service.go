package services

import (
	"context"
	"errors"
	"sync"

	"docflow/internal/models"
	"docflow/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("неверный логин или пароль")
	ErrUserNotActive      = errors.New("пользователь деактивирован")
	ErrNotAuthenticated   = errors.New("требуется авторизация")
	ErrWrongPassword      = errors.New("неверный текущий пароль")
)

type AuthService struct {
	ctx         context.Context
	userRepo    *repository.UserRepository
	currentUser *models.User
	mu          sync.RWMutex
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}

// SetContext вызывается из OnStartup для сохранения контекста Wails
func (s *AuthService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// Login — вход пользователя (Wails binding)
func (s *AuthService) Login(login, password string) (*models.User, error) {
	user, err := s.userRepo.GetByLogin(login)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if !repository.VerifyPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	s.mu.Lock()
	s.currentUser = user
	s.mu.Unlock()

	return user, nil
}

// Logout — выход
func (s *AuthService) Logout() error {
	s.mu.Lock()
	s.currentUser = nil
	s.mu.Unlock()
	return nil
}

// GetCurrentUser — получить текущего пользователя
func (s *AuthService) GetCurrentUser() (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.currentUser == nil {
		return nil, ErrNotAuthenticated
	}

	return s.currentUser, nil
}

// ChangePassword — смена пароля
func (s *AuthService) ChangePassword(oldPassword, newPassword string) error {
	s.mu.RLock()
	user := s.currentUser
	s.mu.RUnlock()

	if user == nil {
		return ErrNotAuthenticated
	}

	dbUser, err := s.userRepo.GetByID(user.ID)
	if err != nil {
		return err
	}

	if !repository.VerifyPassword(dbUser.PasswordHash, oldPassword) {
		return ErrWrongPassword
	}

	newHash, err := repository.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(user.ID, newHash)
}

// IsAuthenticated — проверка авторизации
func (s *AuthService) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentUser != nil
}

// GetCurrentUserID — получить ID текущего пользователя
func (s *AuthService) GetCurrentUserID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentUser == nil {
		return ""
	}
	return s.currentUser.ID.String()
}

// HasRole — проверка роли
func (s *AuthService) HasRole(role string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.currentUser == nil {
		return false
	}

	for _, r := range s.currentUser.Roles {
		if r == role {
			return true
		}
	}
	return false
}

package services

import (
	"context"
	"errors"
	"fmt"
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

// NeedsInitialSetup — проверяет, нужна ли первоначальная настройка (нет пользователей в БД)
func (s *AuthService) NeedsInitialSetup() (bool, error) {
	count, err := s.userRepo.CountUsers()
	if err != nil {
		return false, fmt.Errorf("ошибка проверки пользователей: %w", err)
	}
	return count == 0, nil
}

// InitialSetup — создаёт администратора при первом запуске (работает только если пользователей в БД нет)
func (s *AuthService) InitialSetup(password string) error {
	count, err := s.userRepo.CountUsers()
	if err != nil {
		return fmt.Errorf("ошибка проверки пользователей: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("начальная настройка уже выполнена")
	}

	if len(password) < 6 {
		return fmt.Errorf("пароль должен содержать минимум 6 символов")
	}

	_, err = s.userRepo.Create(models.CreateUserRequest{
		Login:    "admin",
		Password: password,
		FullName: "Администратор",
		Roles:    []string{"admin"},
	})
	if err != nil {
		return fmt.Errorf("ошибка создания администратора: %w", err)
	}

	return nil
}

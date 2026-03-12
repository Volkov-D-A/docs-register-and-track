package services

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// UserService предоставляет бизнес-логику для управления пользователями.
type UserService struct {
	userRepo     UserStore
	auth         *AuthService
	auditService *AdminAuditLogService
}

// NewUserService создает новый экземпляр UserService.
func NewUserService(userRepo UserStore, auth *AuthService, auditService *AdminAuditLogService) *UserService {
	return &UserService{
		userRepo:     userRepo,
		auth:         auth,
		auditService: auditService,
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
	if err != nil {
		return nil, err
	}

	userID, userName := s.getCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "USER_CREATE", fmt.Sprintf("Создан пользователь «%s» (%s)", req.FullName, req.Login))

	return dto.MapUser(res), nil
}

// UpdateUser обновляет данные пользователя (доступно только администраторам).
func (s *UserService) UpdateUser(req models.UpdateUserRequest) (*dto.User, error) {
	if !s.auth.HasRole("admin") {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.Update(req)
	if err != nil {
		return nil, err
	}

	userID, userName := s.getCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "USER_UPDATE", fmt.Sprintf("Обновлен пользователь «%s»", req.FullName))

	return dto.MapUser(res), nil
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

	if err := s.userRepo.ResetPassword(uid, newPassword); err != nil {
		return err
	}

	adminID, adminName := s.getCurrentAuditInfo()
	s.auditService.LogAction(adminID, adminName, "USER_PASSWORD_RESET", fmt.Sprintf("Сброшен пароль пользователя (ID: %s)", userID))
	return nil
}

// GetExecutors возвращает список пользователей-исполнителей.
func (s *UserService) GetExecutors() ([]dto.User, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	res, err := s.userRepo.GetExecutors()
	return dto.MapUsers(res), err
}

// getCurrentAuditInfo возвращает ID и имя текущего пользователя для аудит-лога.
func (s *UserService) getCurrentAuditInfo() (uuid.UUID, string) {
	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return uuid.Nil, "system"
	}
	return uuid.MustParse(user.ID), user.FullName
}

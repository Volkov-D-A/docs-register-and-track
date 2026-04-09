package services

import (
	"fmt"

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
	if err := s.auth.RequireAnyActiveRole("admin", "clerk"); err != nil {
		return nil, err
	}
	res, err := s.userRepo.GetAll()
	return dto.MapUsers(res), err
}

// CreateUser создает нового пользователя (доступно только администраторам).
func (s *UserService) CreateUser(req models.CreateUserRequest) (*dto.User, error) {
	if err := s.auth.RequireActiveRole("admin"); err != nil {
		return nil, err
	}
	res, err := s.userRepo.Create(req)
	if err != nil {
		return nil, err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "USER_CREATE", fmt.Sprintf("Создан пользователь «%s» (%s)", req.FullName, req.Login))

	return dto.MapUser(res), nil
}

// UpdateUser обновляет данные пользователя (доступно только администраторам).
func (s *UserService) UpdateUser(req models.UpdateUserRequest) (*dto.User, error) {
	if err := s.auth.RequireActiveRole("admin"); err != nil {
		return nil, err
	}
	res, err := s.userRepo.Update(req)
	if err != nil {
		return nil, err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "USER_UPDATE", fmt.Sprintf("Обновлен пользователь «%s»", req.FullName))

	return dto.MapUser(res), nil
}

// ResetPassword сбрасывает пароль пользователя (доступно только администраторам).
func (s *UserService) ResetPassword(userID string, newPassword string) error {
	if err := s.auth.RequireActiveRole("admin"); err != nil {
		return err
	}

	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(uid)
	if err != nil {
		return err
	}

	if err := s.userRepo.ResetPassword(uid, newPassword); err != nil {
		return err
	}

	targetUserName := user.FullName
	if targetUserName == "" {
		targetUserName = user.Login
	}

	adminID, adminName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(adminID, adminName, "USER_PASSWORD_RESET", fmt.Sprintf("Сброшен пароль пользователя «%s»", targetUserName))
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

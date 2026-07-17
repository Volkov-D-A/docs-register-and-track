package services

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

// UserService предоставляет бизнес-логику для управления пользователями.
type UserService struct {
	userRepo     UserStore
	auth         *AuthService
	auditService *AdminAuditLogService
}
type userOutboxStore interface {
	CreateWithOutbox(models.CreateUserRequest, []models.OutboxEvent) (*models.User, error)
	UpdateWithOutbox(models.UpdateUserRequest, []models.OutboxEvent) (*models.User, error)
	ResetPasswordWithOutbox(uuid.UUID, string, []models.OutboxEvent) error
}

var errUserOutboxStoreRequired = fmt.Errorf("user store must support atomic outbox operations")

func (s *UserService) auditEffect(key, action, details string) (models.OutboxEvent, error) {
	userID, userName := s.auth.GetCurrentAuditInfo()
	return NewAdminAuditOutboxEvent(key, models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: action, Details: details})
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
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	res, err := s.userRepo.GetAll()
	return dto.MapUsers(res), err
}

// CreateUser создает нового пользователя (доступно только администраторам).
func (s *UserService) CreateUser(req models.CreateUserRequest) (*dto.User, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}

	temporaryPassword := ""
	if req.Password == "" {
		generatedPassword, err := security.GenerateTemporaryPassword()
		if err != nil {
			return nil, err
		}
		req.Password = generatedPassword
		temporaryPassword = generatedPassword
	}
	req.PasswordChangeRequired = true

	details := fmt.Sprintf("Создан пользователь «%s» (%s)", req.FullName, req.Login)
	var res *models.User
	var err error
	store, ok := s.userRepo.(userOutboxStore)
	if !ok {
		return nil, errUserOutboxStoreRequired
	}
	event, buildErr := s.auditEffect("user:"+uuid.NewString()+":create", "USER_CREATE", details)
	if buildErr != nil {
		return nil, buildErr
	}
	res, err = store.CreateWithOutbox(req, []models.OutboxEvent{event})
	if err != nil {
		return nil, err
	}

	userDTO := dto.MapUser(res)
	userDTO.TemporaryPassword = temporaryPassword
	return userDTO, nil
}

// UpdateUser обновляет данные пользователя (доступно только администраторам).
func (s *UserService) UpdateUser(req models.UpdateUserRequest) (*dto.User, error) {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	details := fmt.Sprintf("Обновлен пользователь «%s»", req.FullName)
	var res *models.User
	var err error
	store, ok := s.userRepo.(userOutboxStore)
	if !ok {
		return nil, errUserOutboxStoreRequired
	}
	event, buildErr := s.auditEffect("user:"+req.ID+":update:"+uuid.NewString(), "USER_UPDATE", details)
	if buildErr != nil {
		return nil, buildErr
	}
	res, err = store.UpdateWithOutbox(req, []models.OutboxEvent{event})
	if err != nil {
		return nil, activeAdministratorInvariantConflict(err)
	}

	return dto.MapUser(res), nil
}

// ResetPassword сбрасывает пароль пользователя (доступно только администраторам).
func (s *UserService) ResetPassword(userID string, newPassword string) error {
	if err := s.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
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
	if user == nil {
		return models.NewNotFound("пользователь не найден")
	}

	targetUserName := user.FullName
	if targetUserName == "" {
		targetUserName = user.Login
	}
	details := fmt.Sprintf("Сброшен пароль пользователя «%s»", targetUserName)
	store, ok := s.userRepo.(userOutboxStore)
	if !ok {
		return errUserOutboxStoreRequired
	}
	event, buildErr := s.auditEffect("user:"+uid.String()+":password-reset:"+uuid.NewString(), "USER_PASSWORD_RESET", details)
	if buildErr != nil {
		return buildErr
	}
	err = store.ResetPasswordWithOutbox(uid, newPassword, []models.OutboxEvent{event})
	if err != nil {
		return err
	}

	return nil
}

// GetExecutors возвращает список активных сотрудников для назначений и ознакомления.
func (s *UserService) GetExecutors() ([]dto.User, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.userRepo.GetExecutors()
	return dto.MapUsers(res), err
}

// GetSubstitutionCandidates возвращает активных пользователей, которых можно выбрать замещающими.
func (s *UserService) GetSubstitutionCandidates() ([]dto.User, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	res, err := s.userRepo.GetActiveUsers()
	return dto.MapUsers(res), err
}

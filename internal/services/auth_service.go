package services

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

var (
	ErrInvalidCredentials = models.ErrInvalidCredentials
	ErrUserNotActive      = models.ErrUserNotActive
	ErrUserLocked         = models.ErrUserLocked
	ErrNotAuthenticated   = models.ErrUnauthorized
	ErrWrongPassword      = models.ErrWrongPassword
)

// AuthService предоставляет бизнес-логику для аутентификации и авторизации пользователей.
type AuthService struct {
	db            *database.DB
	userRepo      UserStore
	accessRepo    DocumentAccessStore
	auditService  *AdminAuditLogService
	currentUserID uuid.UUID
	mu            sync.RWMutex
}

// NewAuthService создает новый экземпляр AuthService.
func NewAuthService(db *database.DB, userRepo UserStore) *AuthService {
	return &AuthService{
		db:       db,
		userRepo: userRepo,
	}
}

// SetAdminAuditLogService подключает журнал аудита администратора к сервису аутентификации.
func (s *AuthService) SetAdminAuditLogService(auditService *AdminAuditLogService) {
	s.auditService = auditService
}

// SetAccessStore подключает источник системных прав пользователя.
func (s *AuthService) SetAccessStore(accessRepo DocumentAccessStore) {
	s.accessRepo = accessRepo
}

// isTableNotExistsError проверяет, является ли ошибка «таблица не существует» (PostgreSQL 42P01).
func isTableNotExistsError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "42P01" // undefined_table
	}
	return false
}

// Login — вход пользователя (Wails binding)
func (s *AuthService) Login(login, password string) (*dto.User, error) {
	user, err := s.userRepo.GetByLogin(login)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if !security.VerifyPassword(user.PasswordHash, password) {
		attempts, isActive, err := s.userRepo.IncrementFailedLoginAttempts(user.ID)
		if err != nil {
			return nil, err
		}
		if !isActive {
			if attempts == 5 {
				s.logUserLock(user)
			}
			return nil, ErrUserLocked
		}
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		if user.FailedLoginAttempts >= 5 {
			return nil, ErrUserLocked
		}
		return nil, ErrUserNotActive
	}

	if user.FailedLoginAttempts > 0 {
		if err := s.userRepo.ResetFailedLoginAttempts(user.ID); err != nil {
			return nil, err
		}
		user.FailedLoginAttempts = 0
	}

	s.mu.Lock()
	s.currentUserID = user.ID
	s.mu.Unlock()

	return dto.MapUser(user), nil
}

func (s *AuthService) logUserLock(user *models.User) {
	if s.auditService == nil || user == nil {
		return
	}

	userName := user.FullName
	if userName == "" {
		userName = user.Login
	}

	s.auditService.LogAction(
		user.ID,
		userName,
		"USER_LOCKED",
		fmt.Sprintf("Пользователь «%s» (%s) автоматически заблокирован после 5 неверных попыток входа", userName, user.Login),
	)
}

// Logout — выход
func (s *AuthService) Logout() error {
	s.mu.Lock()
	s.currentUserID = uuid.Nil
	s.mu.Unlock()
	return nil
}

// GetCurrentUser — получить текущего пользователя
func (s *AuthService) GetCurrentUser() (*dto.User, error) {
	s.mu.RLock()
	userID := s.currentUserID
	s.mu.RUnlock()

	if userID == uuid.Nil {
		return nil, ErrNotAuthenticated
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotAuthenticated
	}

	return dto.MapUser(user), nil
}

// GetCurrentUserUUID возвращает ID текущего пользователя, безопасно копируя его под блокировкой.
// Возвращает uuid.Nil и ошибку, если пользователь не авторизован.
func (s *AuthService) GetCurrentUserUUID() (uuid.UUID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentUserID == uuid.Nil {
		return uuid.Nil, ErrNotAuthenticated
	}
	return s.currentUserID, nil
}

// ChangePassword — смена пароля
func (s *AuthService) ChangePassword(oldPassword, newPassword string) error {
	userID, err := s.GetCurrentUserUUID()
	if err != nil {
		return err
	}

	dbUser, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if dbUser == nil {
		return ErrNotAuthenticated
	}

	if !security.VerifyPassword(dbUser.PasswordHash, oldPassword) {
		return ErrWrongPassword
	}

	if err := security.ValidatePassword(newPassword); err != nil {
		return err
	}

	newHash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, newHash)
}

// UpdateProfile — обновление профиля текущего пользователя
func (s *AuthService) UpdateProfile(req models.UpdateProfileRequest) error {
	userID, err := s.GetCurrentUserUUID()
	if err != nil {
		return err
	}

	return s.userRepo.UpdateProfile(userID, req)
}

// IsAuthenticated — проверка авторизации
func (s *AuthService) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentUserID != uuid.Nil
}

// GetCurrentUserID — получить ID текущего пользователя
func (s *AuthService) GetCurrentUserID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentUserID == uuid.Nil {
		return ""
	}
	return s.currentUserID.String()
}

// GetCurrentAuditInfo возвращает ID и имя текущего пользователя для аудит-лога.
// Безопасен: при ошибке возвращает uuid.Nil/"system".
func (s *AuthService) GetCurrentAuditInfo() (uuid.UUID, string) {
	s.mu.RLock()
	userID := s.currentUserID
	s.mu.RUnlock()

	if userID == uuid.Nil {
		return uuid.Nil, "system"
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID, "system"
	}
	if user == nil {
		return userID, "system"
	}

	return user.ID, user.FullName
}

// HasSystemPermission проверяет прямое системное право текущего пользователя.
func (s *AuthService) HasSystemPermission(permission string) bool {
	s.mu.RLock()
	userID := s.currentUserID
	s.mu.RUnlock()

	if userID == uuid.Nil || s.accessRepo == nil {
		return false
	}

	allowed, err := s.accessRepo.HasSystemPermission(permission, userID.String())
	if err != nil {
		return false
	}
	return allowed
}

// RequireSystemPermission возвращает nil если у текущего пользователя есть указанное системное право.
func (s *AuthService) RequireSystemPermission(permission string) error {
	if !s.IsAuthenticated() {
		return models.ErrUnauthorized
	}
	if !s.HasSystemPermission(permission) {
		return models.ErrForbidden
	}
	return nil
}

// HasAnySystemPermission проверяет наличие хотя бы одного системного права у текущего пользователя.
func (s *AuthService) HasAnySystemPermission(permissions ...string) bool {
	for _, permission := range permissions {
		if s.HasSystemPermission(permission) {
			return true
		}
	}
	return false
}

// RequireAnySystemPermission возвращает nil если у текущего пользователя есть хотя бы одно системное право.
func (s *AuthService) RequireAnySystemPermission(permissions ...string) error {
	if !s.IsAuthenticated() {
		return models.ErrUnauthorized
	}
	if !s.HasAnySystemPermission(permissions...) {
		return models.ErrForbidden
	}
	return nil
}

// NeedsInitialSetup — проверяет, нужна ли первоначальная настройка.
// Возвращает true если таблицы ещё не созданы (миграции не применены) или пользователей в БД нет.
func (s *AuthService) NeedsInitialSetup() (bool, error) {
	count, err := s.userRepo.CountUsers()
	if err != nil {
		if isTableNotExistsError(err) {
			// Таблицы ещё не созданы — нужна первоначальная настройка (включая миграции)
			return true, nil
		}
		return false, fmt.Errorf("ошибка проверки пользователей: %w", err)
	}
	return count == 0, nil
}

// InitialSetup — создаёт администратора при первом запуске.
// Если таблицы не существуют, автоматически запускает миграции перед созданием пользователя.
func (s *AuthService) InitialSetup(password string) error {
	// Проверяем, существуют ли таблицы; если нет — запускаем миграции
	_, err := s.userRepo.CountUsers()
	if err != nil {
		if isTableNotExistsError(err) {
			if migErr := s.db.RunMigrations(database.DefaultMigrationsPath); migErr != nil {
				return fmt.Errorf("ошибка применения миграций: %w", migErr)
			}
		} else {
			return fmt.Errorf("ошибка проверки пользователей: %w", err)
		}
	}

	// После миграций повторно проверяем — вдруг пользователи уже есть
	count, err := s.userRepo.CountUsers()
	if err != nil {
		return fmt.Errorf("ошибка проверки пользователей: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("начальная настройка уже выполнена")
	}

	if err := security.ValidatePassword(password); err != nil {
		return err
	}

	user, err := s.userRepo.Create(models.CreateUserRequest{
		Login:                 "admin",
		Password:              password,
		FullName:              "Администратор",
		IsDocumentParticipant: false,
	})
	if err != nil {
		return fmt.Errorf("ошибка создания администратора: %w", err)
	}

	if s.accessRepo == nil {
		return fmt.Errorf("ошибка назначения системного права администратора: access store не подключен")
	}
	if err := s.accessRepo.ReplaceUserAccessProfile(
		user.ID.String(),
		[]models.UserSystemPermissionRule{{Permission: models.SystemPermissionAdmin, IsAllowed: true}},
		nil,
	); err != nil {
		return fmt.Errorf("ошибка назначения системного права администратора: %w", err)
	}

	return nil
}

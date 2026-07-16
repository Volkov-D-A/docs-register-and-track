package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

var (
	ErrInvalidCredentials     = models.ErrInvalidCredentials
	ErrUserNotActive          = models.ErrUserNotActive
	ErrUserLocked             = models.ErrUserLocked
	ErrPasswordChangeRequired = models.ErrPasswordChangeRequired
	ErrNotAuthenticated       = models.ErrUnauthorized
	ErrWrongPassword          = models.ErrWrongPassword
)

// AuthService предоставляет бизнес-логику для аутентификации и авторизации пользователей.
type AuthService struct {
	db            *database.DB
	userRepo      UserStore
	settingsRepo  SettingsStore
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

// SetSettingsStore подключает источник системных настроек.
func (s *AuthService) SetSettingsStore(settingsRepo SettingsStore) {
	s.settingsRepo = settingsRepo
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
	if err := s.ensureCompatibleSchema(); err != nil {
		return nil, err
	}

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

	if s.isPasswordChangeRequired(user) {
		return nil, ErrPasswordChangeRequired
	}

	s.mu.Lock()
	s.currentUserID = user.ID
	s.mu.Unlock()

	return dto.MapUser(user), nil
}

func (s *AuthService) isPasswordChangeRequired(user *models.User) bool {
	if user == nil {
		return false
	}
	if user.PasswordChangeRequired {
		return true
	}

	lifetimeDays := s.getPasswordLifetimeDays()
	if lifetimeDays <= 0 {
		return false
	}
	if user.PasswordChangedAt == nil {
		return true
	}

	expiresAt := user.PasswordChangedAt.AddDate(0, 0, lifetimeDays)
	return !time.Now().Before(expiresAt)
}

func (s *AuthService) getPasswordLifetimeDays() int {
	if s.settingsRepo == nil {
		return 0
	}

	setting, err := s.settingsRepo.Get("password_lifetime_days")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		return 0
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return 0
	}

	days, err := strconv.Atoi(strings.TrimSpace(setting.Value))
	if err != nil || days < 0 {
		return 0
	}
	return days
}

func (s *AuthService) ensureCompatibleSchema() error {
	if s.db == nil {
		return nil
	}
	if err := s.db.CheckMigrationCompatibility(database.DefaultMigrationsPath); err != nil {
		return migrationCompatibilityAppError(err)
	}
	return nil
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
	user, err := s.getActiveCurrentUser()
	if err != nil {
		return nil, err
	}
	return dto.MapUser(user), nil
}

// GetCurrentUserUUID возвращает ID активного текущего пользователя.
// При удалении или деактивации пользователя локальная сессия отзывается.
func (s *AuthService) GetCurrentUserUUID() (uuid.UUID, error) {
	user, err := s.getActiveCurrentUser()
	if err != nil {
		return uuid.Nil, err
	}
	return user.ID, nil
}

// RequireAuthenticated проверяет, что текущая сессия принадлежит активному пользователю.
// Используется на границах защищённых операций, где недостаточно наличия UUID в памяти.
func (s *AuthService) RequireAuthenticated() error {
	_, err := s.getActiveCurrentUser()
	return err
}

func (s *AuthService) getActiveCurrentUser() (*models.User, error) {
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
	if user == nil || !user.IsActive {
		s.mu.Lock()
		if s.currentUserID == userID {
			s.currentUserID = uuid.Nil
		}
		s.mu.Unlock()
		return nil, ErrNotAuthenticated
	}

	return user, nil
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
		return wrapPasswordPolicyError(err)
	}

	newHash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, newHash)
}

// ChangeRequiredPassword меняет пароль до полноценного входа, когда пароль истек или требуется первичная смена.
func (s *AuthService) ChangeRequiredPassword(login, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByLogin(login)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrInvalidCredentials
	}
	if !user.IsActive {
		if user.FailedLoginAttempts >= 5 {
			return ErrUserLocked
		}
		return ErrUserNotActive
	}

	if !security.VerifyPassword(user.PasswordHash, oldPassword) {
		attempts, isActive, err := s.userRepo.IncrementFailedLoginAttempts(user.ID)
		if err != nil {
			return err
		}
		if !isActive {
			if attempts == 5 {
				s.logUserLock(user)
			}
			return ErrUserLocked
		}
		return ErrInvalidCredentials
	}

	if !s.isPasswordChangeRequired(user) {
		return models.NewConflict("смена пароля сейчас не требуется")
	}
	if err := security.ValidatePassword(newPassword); err != nil {
		return wrapPasswordPolicyError(err)
	}

	newHash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(user.ID, newHash)
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
	userID, err := s.GetCurrentUserUUID()
	if err != nil || s.accessRepo == nil {
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
	if err := s.RequireAuthenticated(); err != nil {
		return err
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
	if err := s.RequireAuthenticated(); err != nil {
		return err
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

	if err := security.ValidatePassword(password); err != nil {
		return wrapPasswordPolicyError(err)
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return fmt.Errorf("ошибка подготовки пароля администратора: %w", err)
	}

	if s.accessRepo == nil {
		return fmt.Errorf("ошибка назначения системного права администратора: access store не подключен")
	}

	bootstrapStore, ok := s.userRepo.(InitialSetupStore)
	if !ok {
		return fmt.Errorf("ошибка создания администратора: хранилище не поддерживает атомарную первоначальную настройку")
	}
	if err := bootstrapStore.CreateInitialAdmin(passwordHash); err != nil {
		if _, ok := models.AsAppError(err); ok {
			return err
		}
		return fmt.Errorf("ошибка создания администратора: %w", err)
	}

	return nil
}

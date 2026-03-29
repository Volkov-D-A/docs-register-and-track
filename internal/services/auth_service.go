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

const migrationsPathAuth = "internal/database/migrations"

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
	auditService  *AdminAuditLogService
	currentUserID uuid.UUID
	activeRole    string
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
	s.activeRole = defaultActiveRole(user.Roles)
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
	s.activeRole = ""
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

func defaultActiveRole(roles []string) string {
	for _, role := range []string{"admin", "clerk", "executor"} {
		for _, userRole := range roles {
			if userRole == role {
				return role
			}
		}
	}

	if len(roles) > 0 {
		return roles[0]
	}

	return ""
}

// SetActiveRole устанавливает активную роль текущего пользователя.
func (s *AuthService) SetActiveRole(role string) error {
	if !s.IsAuthenticated() {
		return models.ErrUnauthorized
	}

	s.mu.RLock()
	userID := s.currentUserID
	s.mu.RUnlock()

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return models.ErrUnauthorized
	}

	for _, userRole := range user.Roles {
		if userRole == role {
			s.mu.Lock()
			s.activeRole = role
			s.mu.Unlock()
			return nil
		}
	}

	return models.ErrForbidden
}

// GetActiveRole возвращает активную роль текущего пользователя.
func (s *AuthService) GetActiveRole() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeRole
}

// HasActiveRole проверяет, совпадает ли активная роль с указанной.
func (s *AuthService) HasActiveRole(role string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentUserID != uuid.Nil && s.activeRole == role
}

// RequireActiveRole возвращает nil, если активная роль совпадает с указанной.
func (s *AuthService) RequireActiveRole(role string) error {
	if !s.IsAuthenticated() {
		return models.ErrUnauthorized
	}
	if !s.HasActiveRole(role) {
		return models.ErrForbidden
	}
	return nil
}

// RequireAnyActiveRole возвращает nil, если активная роль входит в указанный список.
func (s *AuthService) RequireAnyActiveRole(roles ...string) error {
	if !s.IsAuthenticated() {
		return models.ErrUnauthorized
	}
	for _, role := range roles {
		if s.HasActiveRole(role) {
			return nil
		}
	}
	return models.ErrForbidden
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

	return user.ID, user.FullName
}

// HasRole — проверка роли
func (s *AuthService) HasRole(role string) bool {
	s.mu.RLock()
	userID := s.currentUserID
	s.mu.RUnlock()

	if userID == uuid.Nil {
		return false
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return false
	}

	for _, r := range user.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// RequireRole возвращает nil если у текущего пользователя есть указанная роль,
// иначе возвращает ErrUnauthorized (не залогинен) или ErrForbidden (нет прав).
func (s *AuthService) RequireRole(role string) error {
	if !s.IsAuthenticated() {
		return models.ErrUnauthorized
	}
	if !s.HasRole(role) {
		return models.ErrForbidden
	}
	return nil
}

// RequireAnyRole возвращает nil если у текущего пользователя есть хотя бы одна из указанных ролей.
func (s *AuthService) RequireAnyRole(roles ...string) error {
	if !s.IsAuthenticated() {
		return models.ErrUnauthorized
	}
	for _, r := range roles {
		if s.HasRole(r) {
			return nil
		}
	}
	return models.ErrForbidden
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
			if migErr := s.db.RunMigrations(migrationsPathAuth); migErr != nil {
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

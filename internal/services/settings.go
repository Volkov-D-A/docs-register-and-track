package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const rollbackMigrationConfirmationPhrase = "ОТКАТ МИГРАЦИИ"

// SettingsService предоставляет бизнес-логику для работы с системными настройками.
type SettingsService struct {
	db              migrationDatabase
	repo            SettingsStore
	authService     *AuthService
	auditService    *AdminAuditLogService
	schemaLifecycle SchemaLifecycle
	migrationMu     sync.Mutex
}

type migrationDatabase interface {
	RunMigrations(string) error
	GetMigrationStatus(string) (*database.MigrationStatus, error)
	RollbackMigration(string) error
}

type settingsOutboxStore interface {
	UpdateWithOutbox(string, string, []models.OutboxEvent) error
}

var errSettingsOutboxStoreRequired = errors.New("settings store must support atomic outbox operations")

// NewSettingsService создает новый экземпляр SettingsService.
func NewSettingsService(db migrationDatabase, repo SettingsStore, authService *AuthService, auditService *AdminAuditLogService) *SettingsService {
	return &SettingsService{
		db:           db,
		repo:         repo,
		authService:  authService,
		auditService: auditService,
	}
}

// GetAll возвращает все системные настройки.
func (s *SettingsService) GetAll() ([]models.SystemSetting, error) {
	if err := s.authService.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	return s.repo.GetAll()
}

// Update обновляет значение настройки по ключу (только для администраторов).
func (s *SettingsService) Update(key, value string) error {
	if err := s.authService.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return err
	}
	if err := validateSystemSettingValue(key, value); err != nil {
		return err
	}

	current, err := s.repo.Get(key)
	if err == nil && current != nil && current.Value == value {
		return nil
	}

	userID, userName := s.authService.GetCurrentAuditInfo()
	details := fmt.Sprintf("Изменена настройка %s: %s", s.getSettingAuditLabel(key, current), value)
	store, ok := s.repo.(settingsOutboxStore)
	if !ok {
		return errSettingsOutboxStoreRequired
	}
	event, buildErr := NewAdminAuditOutboxEvent("setting:"+key+":update:"+uuid.NewString(), models.CreateAdminAuditLogRequest{UserID: userID, UserName: userName, Action: "SETTINGS_UPDATE", Details: details})
	if buildErr != nil {
		return buildErr
	}
	err = store.UpdateWithOutbox(key, value, []models.OutboxEvent{event})
	if err != nil {
		return err
	}
	return nil
}

func validateSystemSettingValue(key, value string) error {
	switch key {
	case "password_lifetime_days":
		days, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || days < 0 {
			return models.NewBadRequest("Срок жизни пароля должен быть целым числом от 0 дней")
		}
	}
	return nil
}

// RunMigrations запускает миграции БД (только admin).
func (s *SettingsService) RunMigrations() error {
	s.migrationMu.Lock()
	defer s.migrationMu.Unlock()

	if err := s.authService.requireSystemPermissionWithoutSchemaCheck(models.SystemPermissionAdmin); err != nil {
		return models.NewForbidden("Недостаточно прав для управления миграциями")
	}
	if err := s.db.RunMigrations(database.DefaultMigrationsPath); err != nil {
		return migrationCompatibilityAppError(err)
	}

	userID, userName := s.authService.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "MIGRATION_RUN", "Применены миграции БД")
	if s.schemaLifecycle != nil {
		s.schemaLifecycle.ReconcileSchema()
	}
	return nil
}

// GetMigrationStatus возвращает текущий статус миграций БД (только admin).
func (s *SettingsService) GetMigrationStatus() (*database.MigrationStatus, error) {
	if err := s.authService.requireSystemPermissionWithoutSchemaCheck(models.SystemPermissionAdmin); err != nil {
		return nil, models.NewForbidden("Недостаточно прав для просмотра статуса миграций")
	}
	return s.db.GetMigrationStatus(database.DefaultMigrationsPath)
}

// RollbackMigration откатывает последнюю миграцию БД (только admin).
func (s *SettingsService) RollbackMigration(req models.RollbackMigrationRequest) error {
	s.migrationMu.Lock()
	defer s.migrationMu.Unlock()

	if err := s.authService.requireSystemPermissionWithoutSchemaCheck(models.SystemPermissionAdmin); err != nil {
		return models.NewForbidden("Недостаточно прав для отката миграций")
	}
	if err := validateRollbackMigrationRequest(req); err != nil {
		return err
	}
	if s.schemaLifecycle != nil {
		if err := s.schemaLifecycle.CheckReady(); err != nil {
			return err
		}
		if err := s.schemaLifecycle.PrepareRollback(); err != nil {
			return models.NewConflictWrapped("Не удалось остановить фоновые процессы перед откатом миграции", err)
		}
	}
	rollbackSucceeded := false
	defer func() {
		if s.schemaLifecycle != nil {
			s.schemaLifecycle.CompleteRollback(rollbackSucceeded)
		}
	}()

	userID, userName := s.authService.GetCurrentAuditInfo()
	s.auditService.LogAction(
		userID,
		userName,
		"MIGRATION_ROLLBACK_REQUESTED",
		fmt.Sprintf("Запрошен откат последней миграции БД; backup: %s", strings.TrimSpace(req.BackupReference)),
	)

	if err := s.db.RollbackMigration(database.DefaultMigrationsPath); err != nil {
		return migrationCompatibilityAppError(err)
	}

	s.auditService.LogAction(userID, userName, "MIGRATION_ROLLBACK", fmt.Sprintf("Откачена последняя миграция БД; backup: %s", strings.TrimSpace(req.BackupReference)))
	rollbackSucceeded = true
	return nil
}

// Вспомогательные методы для других сервисов

// GetMaxFileSize возвращает максимальный допустимый размер файла в байтах.
func (s *SettingsService) GetMaxFileSize() (int64, error) {
	setting, err := s.repo.Get("max_file_size_mb")
	if err != nil {
		return 15 * 1024 * 1024, nil
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return 15 * 1024 * 1024, nil
	}
	mb, err := strconv.Atoi(setting.Value)
	if err != nil {
		return 15 * 1024 * 1024, nil
	}
	return int64(mb) * 1024 * 1024, nil
}

// GetAllowedFileTypes возвращает список разрешенных расширений файлов.
func (s *SettingsService) GetAllowedFileTypes() ([]string, error) {
	setting, err := s.repo.Get("allowed_file_types")
	if err != nil {
		return []string{".pdf", ".doc", ".docx", ".odt", ".xls", ".xlsx", ".ods"}, nil
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return []string{".pdf", ".doc", ".docx", ".odt", ".xls", ".xlsx", ".ods"}, nil
	}
	types := strings.Split(setting.Value, ",")
	result := make([]string, 0, len(types))
	for i, t := range types {
		types[i] = strings.TrimSpace(strings.ToLower(t))
		if types[i] != "" {
			result = append(result, types[i])
		}
	}
	return result, nil
}

// GetOrganizationName возвращает название основной организации из настроек.
func (s *SettingsService) GetOrganizationName() string {
	setting, err := s.repo.Get("organization_name")
	if err != nil || setting == nil || setting.Value == "" {
		return ""
	}
	return setting.Value
}

// GetOrganizationShortName возвращает краткое название организации из настроек.
func (s *SettingsService) GetOrganizationShortName() string {
	setting, err := s.repo.Get("organization_short_name")
	if err != nil || setting == nil || setting.Value == "" {
		return ""
	}
	return setting.Value
}

// IsAssignmentCompletionAttachmentsEnabled возвращает признак доступности загрузки файлов при завершении поручения.
func (s *SettingsService) IsAssignmentCompletionAttachmentsEnabled() bool {
	setting, err := s.repo.Get("assignment_completion_attachments_enabled")
	if err != nil || setting == nil || setting.Value == "" {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(setting.Value)) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

func validateRollbackMigrationRequest(req models.RollbackMigrationRequest) error {
	if !req.BackupCompleted {
		return models.NewBadRequest("Перед откатом миграции подтвердите свежую резервную копию PostgreSQL и MinIO")
	}
	if strings.TrimSpace(req.BackupReference) == "" {
		return models.NewBadRequest("Укажите идентификатор или путь к резервной копии перед откатом миграции")
	}
	if !req.AcknowledgedDataLoss {
		return models.NewBadRequest("Подтвердите, что откат миграции может удалить данные")
	}
	if strings.TrimSpace(req.Confirmation) != rollbackMigrationConfirmationPhrase {
		return models.NewBadRequest("Введите контрольную фразу для отката миграции")
	}
	return nil
}

func migrationCompatibilityAppError(err error) error {
	var compatibilityErr *database.MigrationCompatibilityError
	if !errors.As(err, &compatibilityErr) {
		return err
	}

	if compatibilityErr.SchemaTooNew {
		return models.NewConflict(fmt.Sprintf(
			"Версия схемы БД (%d) новее миграций, встроенных в приложение (%d). Запустите совместимую версию приложения или выполните утвержденную процедуру обновления.",
			compatibilityErr.CurrentVersion,
			compatibilityErr.LatestAvailableVersion,
		))
	}
	if compatibilityErr.Dirty {
		return models.NewConflict(fmt.Sprintf(
			"Миграция БД версии %d завершилась с ошибкой. Работа заблокирована до восстановления схемы по регламенту.",
			compatibilityErr.CurrentVersion,
		))
	}

	return models.NewConflict("Схема БД несовместима с текущей версией приложения")
}

func (s *SettingsService) getSettingAuditLabel(key string, current *models.SystemSetting) string {
	switch key {
	case "organization_name":
		return "Название организации"
	case "organization_short_name":
		return "Краткое название организации"
	case "max_file_size_mb":
		return "Максимальный размер файла"
	case "allowed_file_types":
		return "Разрешенные типы файлов"
	case "assignment_completion_attachments_enabled":
		return "Файлы при завершении поручения"
	case "password_lifetime_days":
		return "Срок жизни пароля"
	}

	if current != nil && strings.TrimSpace(current.Description) != "" {
		return current.Description
	}

	return fmt.Sprintf("«%s»", key)
}

package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

const rollbackMigrationConfirmationPhrase = "ОТКАТ МИГРАЦИИ"

// SettingsService предоставляет бизнес-логику для работы с системными настройками.
type SettingsService struct {
	repo            SettingsStore
	authService     *AuthService
	schemaLifecycle SchemaLifecycle
	migrationClient serverclient.MigrationClient
	migrationMu     sync.Mutex
}

type settingsOutboxStore interface {
	UpdateWithOutbox(string, string, []models.OutboxEvent) error
}

var errSettingsOutboxStoreRequired = errors.New("settings store must support atomic outbox operations")

// NewSettingsService создает новый экземпляр SettingsService.
func NewSettingsService(repo SettingsStore, authService *AuthService) *SettingsService {
	return &SettingsService{
		repo:        repo,
		authService: authService,
	}
}

func (s *SettingsService) SetMigrationClient(client serverclient.MigrationClient) {
	s.migrationClient = client
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
func (s *SettingsService) RunMigrations(password string) error {
	s.migrationMu.Lock()
	defer s.migrationMu.Unlock()

	if err := s.authService.requireSystemPermissionWithoutSchemaCheck(models.SystemPermissionAdmin); err != nil {
		return models.NewForbidden("Недостаточно прав для управления миграциями")
	}
	login, err := s.currentMigrationLogin(password)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := s.migrationClient.Apply(ctx, login, password); err != nil {
		return models.NewConflictWrapped("Не удалось применить миграции через docflow-server", err)
	}
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
	if s.migrationClient == nil {
		return nil, models.NewConflict("Клиент docflow-server не настроен")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	status, err := s.migrationClient.Status(ctx)
	if err != nil {
		return nil, models.NewConflictWrapped("Не удалось получить статус миграций от docflow-server", err)
	}
	return status, nil
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
	login, err := s.currentMigrationLogin(req.Password)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := s.migrationClient.Rollback(ctx, login, req.Password, req); err != nil {
		return models.NewConflictWrapped("Не удалось откатить миграцию через docflow-server", err)
	}
	if s.schemaLifecycle != nil {
		s.schemaLifecycle.ReconcileSchema()
	}
	return nil
}

func (s *SettingsService) currentMigrationLogin(password string) (string, error) {
	if s.migrationClient == nil {
		return "", models.NewConflict("Клиент docflow-server не настроен")
	}
	if strings.TrimSpace(password) == "" {
		return "", models.NewBadRequest("Введите пароль администратора")
	}
	user, err := s.authService.GetCurrentUser()
	if err != nil {
		return "", err
	}
	return user.Login, nil
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

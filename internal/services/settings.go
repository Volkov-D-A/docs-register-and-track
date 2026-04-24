package services

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// SettingsService предоставляет бизнес-логику для работы с системными настройками.
type SettingsService struct {
	db           *database.DB
	repo         SettingsStore
	authService  *AuthService
	auditService *AdminAuditLogService
}

// NewSettingsService создает новый экземпляр SettingsService.
func NewSettingsService(db *database.DB, repo SettingsStore, authService *AuthService, auditService *AdminAuditLogService) *SettingsService {
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

	current, err := s.repo.Get(key)
	if err == nil && current != nil && current.Value == value {
		return nil
	}

	if err := s.repo.Update(key, value); err != nil {
		return err
	}

	userID, userName := s.authService.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "SETTINGS_UPDATE", fmt.Sprintf("Изменена настройка %s: %s", s.getSettingAuditLabel(key, current), value))
	return nil
}

// RunMigrations запускает миграции БД (только admin).
func (s *SettingsService) RunMigrations() error {
	if err := s.authService.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return models.NewForbidden("Недостаточно прав для управления миграциями")
	}
	if err := s.db.RunMigrations(database.DefaultMigrationsPath); err != nil {
		return err
	}

	userID, userName := s.authService.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "MIGRATION_RUN", "Применены миграции БД")
	return nil
}

// GetMigrationStatus возвращает текущий статус миграций БД (только admin).
func (s *SettingsService) GetMigrationStatus() (*database.MigrationStatus, error) {
	if err := s.authService.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, models.NewForbidden("Недостаточно прав для просмотра статуса миграций")
	}
	return s.db.GetMigrationStatus(database.DefaultMigrationsPath)
}

// RollbackMigration откатывает последнюю миграцию БД (только admin).
func (s *SettingsService) RollbackMigration() error {
	if err := s.authService.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return models.NewForbidden("Недостаточно прав для отката миграций")
	}
	if err := s.db.RollbackMigration(database.DefaultMigrationsPath); err != nil {
		return err
	}

	userID, userName := s.authService.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "MIGRATION_ROLLBACK", "Откачена последняя миграция БД")
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
	}

	if current != nil && strings.TrimSpace(current.Description) != "" {
		return current.Description
	}

	return fmt.Sprintf("«%s»", key)
}

package services

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const migrationsPath = "internal/database/migrations"

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
	if err := s.authService.RequireActiveRole("admin"); err != nil {
		return nil, err
	}
	return s.repo.GetAll()
}

// Update обновляет значение настройки по ключу (только для администраторов).
func (s *SettingsService) Update(key, value string) error {
	if err := s.authService.RequireActiveRole("admin"); err != nil {
		return err
	}
	if err := s.repo.Update(key, value); err != nil {
		return err
	}

	userID, userName := s.authService.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "SETTINGS_UPDATE", fmt.Sprintf("Изменена настройка «%s»: %s", key, value))
	return nil
}

// RunMigrations запускает миграции БД (только admin).
func (s *SettingsService) RunMigrations() error {
	if err := s.authService.RequireActiveRole("admin"); err != nil {
		return models.NewForbidden("Недостаточно прав для управления миграциями")
	}
	if err := s.db.RunMigrations(migrationsPath); err != nil {
		return err
	}

	userID, userName := s.authService.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "MIGRATION_RUN", "Применены миграции БД")
	return nil
}

// GetMigrationStatus возвращает текущий статус миграций БД (только admin).
func (s *SettingsService) GetMigrationStatus() (*database.MigrationStatus, error) {
	if err := s.authService.RequireActiveRole("admin"); err != nil {
		return nil, models.NewForbidden("Недостаточно прав для просмотра статуса миграций")
	}
	return s.db.GetMigrationStatus(migrationsPath)
}

// RollbackMigration откатывает последнюю миграцию БД (только admin).
func (s *SettingsService) RollbackMigration() error {
	if err := s.authService.RequireActiveRole("admin"); err != nil {
		return models.NewForbidden("Недостаточно прав для отката миграций")
	}
	if err := s.db.RollbackMigration(migrationsPath); err != nil {
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
		return 10 * 1024 * 1024, nil // По умолчанию 10 МБ
	}
	mb, err := strconv.Atoi(setting.Value)
	if err != nil {
		return 10 * 1024 * 1024, nil
	}
	return int64(mb) * 1024 * 1024, nil
}

// GetAllowedFileTypes возвращает список разрешенных расширений файлов.
func (s *SettingsService) GetAllowedFileTypes() ([]string, error) {
	setting, err := s.repo.Get("allowed_file_types")
	if err != nil {
		return []string{".pdf", ".doc", ".docx", ".jpg", ".png"}, nil
	}
	types := strings.Split(setting.Value, ",")
	for i, t := range types {
		types[i] = strings.TrimSpace(strings.ToLower(t))
	}
	return types, nil
}

// GetOrganizationName возвращает название основной организации из настроек.
func (s *SettingsService) GetOrganizationName() string {
	setting, err := s.repo.Get("organization_name")
	if err != nil || setting.Value == "" {
		return "НАША ОРГАНИЗАЦИЯ"
	}
	return setting.Value
}

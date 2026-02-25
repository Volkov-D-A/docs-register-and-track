package services

import (
	"docflow/internal/database"
	"docflow/internal/models"
	"strconv"
	"strings"
)

const migrationsPath = "internal/database/migrations"

// SettingsService предоставляет бизнес-логику для работы с системными настройками.
type SettingsService struct {
	db          *database.DB
	repo        SettingsStore
	authService *AuthService
}

// NewSettingsService создает новый экземпляр SettingsService.
func NewSettingsService(db *database.DB, repo SettingsStore, authService *AuthService) *SettingsService {
	return &SettingsService{
		db:          db,
		repo:        repo,
		authService: authService,
	}
}

// GetAll возвращает все системные настройки.
func (s *SettingsService) GetAll() ([]models.SystemSetting, error) {
	// Пока разрешаем чтение всем авторизованным пользователям
	return s.repo.GetAll()
}

// Update обновляет значение настройки по ключу (только для администраторов).
func (s *SettingsService) Update(key, value string) error {
	if !s.authService.HasRole("admin") {
		return models.ErrForbidden
	}
	return s.repo.Update(key, value)
}

// RunMigrations запускает миграции БД (только admin).
func (s *SettingsService) RunMigrations() error {
	if !s.authService.HasRole("admin") {
		return models.NewForbidden("Недостаточно прав для управления миграциями")
	}
	return s.db.RunMigrations(migrationsPath)
}

// GetMigrationStatus возвращает текущий статус миграций БД (только admin).
func (s *SettingsService) GetMigrationStatus() (*database.MigrationStatus, error) {
	if !s.authService.HasRole("admin") {
		return nil, models.NewForbidden("Недостаточно прав для просмотра статуса миграций")
	}
	return s.db.GetMigrationStatus(migrationsPath)
}

// RollbackMigration откатывает последнюю миграцию БД (только admin).
func (s *SettingsService) RollbackMigration() error {
	if !s.authService.HasRole("admin") {
		return models.NewForbidden("Недостаточно прав для отката миграций")
	}
	return s.db.RollbackMigration(migrationsPath)
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

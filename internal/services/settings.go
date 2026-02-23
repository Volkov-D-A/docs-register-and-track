package services

import (
	"context"
	"docflow/internal/database"
	"docflow/internal/models"
	"docflow/internal/repository"
	"strconv"
	"strings"
)

const migrationsPath = "internal/database/migrations"

type SettingsService struct {
	ctx         context.Context
	db          *database.DB
	repo        *repository.SettingsRepository
	authService *AuthService
}

func NewSettingsService(db *database.DB, repo *repository.SettingsRepository, authService *AuthService) *SettingsService {
	return &SettingsService{
		db:          db,
		repo:        repo,
		authService: authService,
	}
}

func (s *SettingsService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// GetAll — получить все системные настройки
func (s *SettingsService) GetAll() ([]models.SystemSetting, error) {
	// Пока разрешаем чтение всем авторизованным пользователям
	return s.repo.GetAll()
}

// Update — обновить значение настройки
func (s *SettingsService) Update(key, value string) error {
	if !s.authService.HasRole("admin") {
		return &models.AppError{Code: 403, Message: "Permission denied"}
	}
	return s.repo.Update(key, value)
}

// RunMigrations запускает миграции БД (только admin).
func (s *SettingsService) RunMigrations() error {
	if !s.authService.HasRole("admin") {
		return &models.AppError{Code: 403, Message: "Недостаточно прав для управления миграциями"}
	}
	return s.db.RunMigrations(migrationsPath)
}

// GetMigrationStatus возвращает текущий статус миграций БД (только admin).
func (s *SettingsService) GetMigrationStatus() (*database.MigrationStatus, error) {
	if !s.authService.HasRole("admin") {
		return nil, &models.AppError{Code: 403, Message: "Недостаточно прав для просмотра статуса миграций"}
	}
	return s.db.GetMigrationStatus(migrationsPath)
}

// RollbackMigration откатывает последнюю миграцию БД (только admin).
func (s *SettingsService) RollbackMigration() error {
	if !s.authService.HasRole("admin") {
		return &models.AppError{Code: 403, Message: "Недостаточно прав для отката миграций"}
	}
	return s.db.RollbackMigration(migrationsPath)
}

// Вспомогательные методы для других сервисов

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

func (s *SettingsService) GetOrganizationName() string {
	setting, err := s.repo.Get("organization_name")
	if err != nil || setting.Value == "" {
		return "НАША ОРГАНИЗАЦИЯ"
	}
	return setting.Value
}

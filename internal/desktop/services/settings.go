package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

const rollbackMigrationConfirmationPhrase = "ОТКАТ МИГРАЦИИ"

type settingsPrincipal interface {
	GetCurrentUser() (*dto.User, error)
	RequireSystemPermissionWithoutSchemaCheck(string) error
}

// SettingsService exposes settings and migration operations through the server API.
type SettingsService struct {
	authService     settingsPrincipal
	schemaLifecycle interface{ ReconcileSchema() }
	migrationClient serverclient.MigrationClient
	settingsClient  serverclient.SettingsClient
	migrationMu     sync.Mutex
}

// NewSettingsService wires the desktop settings and migration clients.
func NewSettingsService(auth settingsPrincipal, settings serverclient.SettingsClient, migrations serverclient.MigrationClient, lifecycle interface{ ReconcileSchema() }) *SettingsService {
	return &SettingsService{authService: auth, settingsClient: settings, migrationClient: migrations, schemaLifecycle: lifecycle}
}

// GetAll возвращает все системные настройки.
func (s *SettingsService) GetAll() ([]models.SystemSetting, error) {
	if s.settingsClient == nil {
		return nil, models.NewConflict("Клиент настроек docflow-server не настроен")
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return s.settingsClient.ListSettings(ctx)
}

// Update обновляет значение настройки по ключу (только для администраторов).
func (s *SettingsService) Update(key, value string) error {
	if s.settingsClient == nil {
		return models.NewConflict("Клиент настроек docflow-server не настроен")
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return s.settingsClient.UpdateSystemSetting(ctx, key, value)
}

// RunMigrations запускает миграции БД (только admin).
func (s *SettingsService) RunMigrations(password string) error {
	s.migrationMu.Lock()
	defer s.migrationMu.Unlock()

	if err := s.authService.RequireSystemPermissionWithoutSchemaCheck(models.SystemPermissionAdmin); err != nil {
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
func (s *SettingsService) GetMigrationStatus() (*dto.MigrationStatus, error) {
	if err := s.authService.RequireSystemPermissionWithoutSchemaCheck(models.SystemPermissionAdmin); err != nil {
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

	if err := s.authService.RequireSystemPermissionWithoutSchemaCheck(models.SystemPermissionAdmin); err != nil {
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

// GetOrganizationShortName возвращает краткое название организации из настроек.
func (s *SettingsService) GetOrganizationShortName() string {
	setting, err := s.getSetting("organization_short_name")
	if err != nil || setting == nil || setting.Value == "" {
		return ""
	}
	return setting.Value
}

// IsAssignmentCompletionAttachmentsEnabled возвращает признак доступности загрузки файлов при завершении поручения.
func (s *SettingsService) IsAssignmentCompletionAttachmentsEnabled() bool {
	setting, err := s.getSetting("assignment_completion_attachments_enabled")
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

func (s *SettingsService) getSetting(key string) (*models.SystemSetting, error) {
	if s.settingsClient == nil {
		return nil, models.NewConflict("Клиент настроек docflow-server не настроен")
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return s.settingsClient.GetSystemSetting(ctx, key)
}

func settingsContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

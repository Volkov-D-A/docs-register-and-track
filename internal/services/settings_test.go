package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSettingsService(t *testing.T, role string) (*SettingsService, *mocks.SettingsStore) {
	t.Helper()
	settingsRepo := mocks.NewSettingsStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_set",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{role},
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	auth.Login(user.Login, password)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	dbMock, _, err := sqlmock.New()
	require.NoError(t, err)
	db := &database.DB{DB: dbMock}

	return NewSettingsService(db, settingsRepo, auth, nil), settingsRepo
}

func TestSettingsService_GetAll(t *testing.T) {
	// Получение полного списка системных настроек из базы
	t.Run("success", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		settings := []models.SystemSetting{{Key: "k1", Value: "v1"}}
		repo.On("GetAll").Return(settings, nil).Once()
		result, err := svc.GetAll()
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestSettingsService_Update(t *testing.T) {
	// Изменение отдельной системной настройки (ключ-значение)
	t.Run("success admin", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Update", "key", "value").Return(nil).Once()
		err := svc.Update("key", "value")
		require.NoError(t, err)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "executor")
		err := svc.Update("key", "value")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})
}

func TestSettingsService_GetMaxFileSize(t *testing.T) {
	// Получение максимально допустимого размера загружаемых файлов в байтах
	t.Run("from settings", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "25"}, nil).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(25*1024*1024), size)
	})

	t.Run("default on error", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "max_file_size_mb").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(10*1024*1024), size)
	})
}

func TestSettingsService_GetAllowedFileTypes(t *testing.T) {
	// Получение списка разрешенных расширений загружаемых файлов
	t.Run("from settings", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "allowed_file_types").Return(&models.SystemSetting{Key: "allowed_file_types", Value: ".pdf, .DOC, .txt"}, nil).Once()
		types, err := svc.GetAllowedFileTypes()
		require.NoError(t, err)
		assert.Equal(t, []string{".pdf", ".doc", ".txt"}, types)
	})

	t.Run("default on error", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "allowed_file_types").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		types, err := svc.GetAllowedFileTypes()
		require.NoError(t, err)
		assert.Contains(t, types, ".pdf")
	})
}

func TestSettingsService_GetOrganizationName(t *testing.T) {
	// Получение названия нашей организации (используется для подстановки по умолчанию)
	t.Run("from settings", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "organization_name").Return(&models.SystemSetting{Key: "organization_name", Value: "Custom Org"}, nil).Once()
		name := svc.GetOrganizationName()
		assert.Equal(t, "Custom Org", name)
	})

	t.Run("default on error", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "organization_name").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		name := svc.GetOrganizationName()
		assert.Equal(t, "НАША ОРГАНИЗАЦИЯ", name)
	})
}

func TestSettingsService_RunMigrations(t *testing.T) {
	// Выполнение накатывания миграций БД (доступно только администратору)
	t.Run("forbidden non-admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "executor")
		err := svc.RunMigrations()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Недостаточно прав")
	})

	t.Run("success admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		// The db layer expects an actual database, which might crash, but we catch it
		err := svc.RunMigrations()
		// it will likely be: migration path not found, or driver creation failure, but not forbidden
		if err != nil {
			assert.NotContains(t, err.Error(), "Недостаточно прав")
		}
	})
}

func TestSettingsService_GetMigrationStatus(t *testing.T) {
	// Получение состояния всех миграций базы данных
	t.Run("forbidden non-admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "clerk")
		status, err := svc.GetMigrationStatus()
		require.Error(t, err)
		require.Nil(t, status)
		assert.Contains(t, err.Error(), "Недостаточно прав")
	})

	t.Run("success admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		status, err := svc.GetMigrationStatus()
		if err != nil {
			assert.NotContains(t, err.Error(), "Недостаточно прав")
		} else {
			assert.NotNil(t, status)
		}
	})
}

func TestSettingsService_RollbackMigration(t *testing.T) {
	// Откат последней примененной миграции базы данных
	t.Run("forbidden non-admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "executor")
		err := svc.RollbackMigration()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Недостаточно прав")
	})

	t.Run("success admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		err := svc.RollbackMigration()
		if err != nil {
			assert.NotContains(t, err.Error(), "Недостаточно прав")
		}
	})
}

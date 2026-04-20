package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"

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
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(role))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_set",
		FullName:     role + " settings",
		PasswordHash: hash,
		IsActive:     true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	auth.Login(user.Login, password)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	dbMock, _, err := sqlmock.New()
	require.NoError(t, err)
	db := &database.DB{DB: dbMock}

	return NewSettingsService(db, settingsRepo, auth, nil), settingsRepo
}

func setupSettingsServiceWithRoles(t *testing.T, roles []string) (*SettingsService, *mocks.SettingsStore, *AuthService, *models.User) {
	t.Helper()
	settingsRepo := mocks.NewSettingsStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(roles...))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        "multi_role_set_" + uuid.New().String(),
		FullName:     "Multi Role Settings",
		PasswordHash: hash,
		IsActive:     true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	auth.Login(user.Login, password)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	dbMock, _, err := sqlmock.New()
	require.NoError(t, err)
	db := &database.DB{DB: dbMock}

	return NewSettingsService(db, settingsRepo, auth, nil), settingsRepo, auth, user
}

type captureSettingsAuditLogStore struct {
	requests []models.CreateAdminAuditLogRequest
}

func (s *captureSettingsAuditLogStore) Create(req models.CreateAdminAuditLogRequest) (uuid.UUID, error) {
	s.requests = append(s.requests, req)
	return uuid.New(), nil
}

func (s *captureSettingsAuditLogStore) GetAll(limit, offset int) ([]models.AdminAuditLog, int, error) {
	return nil, 0, nil
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

	t.Run("allowed for user with admin role", func(t *testing.T) {
		svc, repo, _, _ := setupSettingsServiceWithRoles(t, []string{"admin", "clerk"})
		repo.On("GetAll").Return([]models.SystemSetting{{Key: "k1", Value: "v1"}}, nil).Once()

		result, err := svc.GetAll()
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestSettingsService_Update(t *testing.T) {
	// Изменение отдельной системной настройки (ключ-значение)
	t.Run("success admin", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "key").Return(&models.SystemSetting{Key: "key", Value: "old"}, nil).Once()
		repo.On("Update", "key", "value").Return(nil).Once()
		err := svc.Update("key", "value")
		require.NoError(t, err)
	})

	t.Run("skips update and audit when value did not change", func(t *testing.T) {
		svc, repo, auth, _ := setupSettingsServiceWithRoles(t, []string{"admin"})
		auditRepo := &captureSettingsAuditLogStore{}
		svc.auditService = NewAdminAuditLogService(auditRepo, auth)

		repo.On("Get", "key").Return(&models.SystemSetting{Key: "key", Value: "value"}, nil).Once()

		err := svc.Update("key", "value")
		require.NoError(t, err)
		assert.Empty(t, auditRepo.requests)
	})

	t.Run("writes audit only when value changed", func(t *testing.T) {
		svc, repo, auth, user := setupSettingsServiceWithRoles(t, []string{"admin"})
		auditRepo := &captureSettingsAuditLogStore{}
		svc.auditService = NewAdminAuditLogService(auditRepo, auth)

		repo.On("Get", "key").Return(&models.SystemSetting{Key: "key", Value: "old"}, nil).Once()
		repo.On("Update", "key", "new").Return(nil).Once()

		err := svc.Update("key", "new")
		require.NoError(t, err)
		require.Len(t, auditRepo.requests, 1)
		assert.Equal(t, user.ID, auditRepo.requests[0].UserID)
		assert.Equal(t, user.FullName, auditRepo.requests[0].UserName)
		assert.Equal(t, "SETTINGS_UPDATE", auditRepo.requests[0].Action)
		assert.Equal(t, "Изменена настройка «key»: new", auditRepo.requests[0].Details)
	})

	t.Run("uses human readable label for known setting", func(t *testing.T) {
		svc, repo, auth, _ := setupSettingsServiceWithRoles(t, []string{"admin"})
		auditRepo := &captureSettingsAuditLogStore{}
		svc.auditService = NewAdminAuditLogService(auditRepo, auth)

		repo.On("Get", "assignment_completion_attachments_enabled").Return(&models.SystemSetting{
			Key:         "assignment_completion_attachments_enabled",
			Value:       "true",
			Description: "Разрешить исполнителю прикладывать файлы при завершении поручения",
		}, nil).Once()
		repo.On("Update", "assignment_completion_attachments_enabled", "false").Return(nil).Once()

		err := svc.Update("assignment_completion_attachments_enabled", "false")
		require.NoError(t, err)
		require.Len(t, auditRepo.requests, 1)
		assert.Equal(t, "Изменена настройка Файлы при завершении поручения: false", auditRepo.requests[0].Details)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "executor")
		err := svc.Update("key", "value")
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("allowed for user with admin role", func(t *testing.T) {
		svc, repo, _, _ := setupSettingsServiceWithRoles(t, []string{"admin", "clerk"})
		repo.On("Get", "key").Return(&models.SystemSetting{Key: "key", Value: "old"}, nil).Once()
		repo.On("Update", "key", "value").Return(nil).Once()

		err := svc.Update("key", "value")
		require.NoError(t, err)
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
		assert.Equal(t, int64(15*1024*1024), size)
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
		assert.Equal(t, []string{".pdf", ".doc", ".docx", ".odt", ".xls", ".xlsx", ".ods"}, types)
	})

	t.Run("empty setting returns default list", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "allowed_file_types").Return(&models.SystemSetting{Key: "allowed_file_types", Value: ""}, nil).Once()
		types, err := svc.GetAllowedFileTypes()
		require.NoError(t, err)
		assert.Equal(t, []string{".pdf", ".doc", ".docx", ".odt", ".xls", ".xlsx", ".ods"}, types)
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
		assert.Equal(t, "", name)
	})
}

func TestSettingsService_GetOrganizationShortName(t *testing.T) {
	t.Run("from settings", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "organization_short_name").Return(&models.SystemSetting{Key: "organization_short_name", Value: "Custom Short"}, nil).Once()
		name := svc.GetOrganizationShortName()
		assert.Equal(t, "Custom Short", name)
	})

	t.Run("default on error", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "organization_short_name").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		name := svc.GetOrganizationShortName()
		assert.Equal(t, "", name)
	})
}

func TestSettingsService_IsAssignmentCompletionAttachmentsEnabled(t *testing.T) {
	t.Run("from settings enabled", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "assignment_completion_attachments_enabled").Return(&models.SystemSetting{
			Key:   "assignment_completion_attachments_enabled",
			Value: "true",
		}, nil).Once()
		assert.True(t, svc.IsAssignmentCompletionAttachmentsEnabled())
	})

	t.Run("from settings disabled", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "assignment_completion_attachments_enabled").Return(&models.SystemSetting{
			Key:   "assignment_completion_attachments_enabled",
			Value: "false",
		}, nil).Once()
		assert.False(t, svc.IsAssignmentCompletionAttachmentsEnabled())
	})

	t.Run("default on error", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "assignment_completion_attachments_enabled").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		assert.False(t, svc.IsAssignmentCompletionAttachmentsEnabled())
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

	t.Run("allowed for user with admin role", func(t *testing.T) {
		svc, _, _, _ := setupSettingsServiceWithRoles(t, []string{"admin", "clerk"})

		err := svc.RunMigrations()
		if err != nil {
			assert.NotContains(t, err.Error(), "Недостаточно прав")
		}
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

	t.Run("allowed for user with admin role", func(t *testing.T) {
		svc, _, _, _ := setupSettingsServiceWithRoles(t, []string{"admin", "clerk"})

		status, err := svc.GetMigrationStatus()
		if err != nil {
			assert.NotContains(t, err.Error(), "Недостаточно прав")
		} else {
			assert.NotNil(t, status)
		}
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

	t.Run("allowed for user with admin role", func(t *testing.T) {
		svc, _, _, _ := setupSettingsServiceWithRoles(t, []string{"admin", "clerk"})

		err := svc.RollbackMigration()
		if err != nil {
			assert.NotContains(t, err.Error(), "Недостаточно прав")
		}
	})

	t.Run("success admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		err := svc.RollbackMigration()
		if err != nil {
			assert.NotContains(t, err.Error(), "Недостаточно прав")
		}
	})
}

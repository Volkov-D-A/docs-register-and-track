package services

import (
	"context"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeServerMigrationClient struct {
	status        database.MigrationStatus
	statusErr     error
	applyErr      error
	rollbackErr   error
	applyCalls    int
	rollbackCalls int
	login         string
	password      string
}

type fakeServerSettingsClient struct {
	store SettingsStore
}

func (c *fakeServerSettingsClient) ListSettings(context.Context) ([]models.SystemSetting, error) {
	return c.store.GetAll()
}

func (c *fakeServerSettingsClient) GetSystemSetting(_ context.Context, key string) (*models.SystemSetting, error) {
	return c.store.Get(key)
}

func (c *fakeServerSettingsClient) UpdateSystemSetting(_ context.Context, key, value string) error {
	return c.store.Update(key, value)
}

func (c *fakeServerMigrationClient) Status(context.Context) (*database.MigrationStatus, error) {
	if c.statusErr != nil {
		return nil, c.statusErr
	}
	return &c.status, nil
}

func (c *fakeServerMigrationClient) Apply(_ context.Context, login, password string) (*database.MigrationStatus, error) {
	c.applyCalls++
	c.login, c.password = login, password
	if c.applyErr != nil {
		return nil, c.applyErr
	}
	return &c.status, nil
}

func (c *fakeServerMigrationClient) Rollback(_ context.Context, login, password string, _ models.RollbackMigrationRequest) (*database.MigrationStatus, error) {
	c.rollbackCalls++
	c.login, c.password = login, password
	if c.rollbackErr != nil {
		return nil, c.rollbackErr
	}
	return &c.status, nil
}

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

	service := NewSettingsService(auth)
	service.SetServerClient(&fakeServerSettingsClient{store: settingsRepo})
	service.SetMigrationClient(&fakeServerMigrationClient{})
	return service, settingsRepo
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

	service := NewSettingsService(auth)
	service.SetServerClient(&fakeServerSettingsClient{store: settingsRepo})
	service.SetMigrationClient(&fakeServerMigrationClient{})
	return service, settingsRepo, auth, user
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

	t.Run("propagates server error", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("GetAll").Return(nil, models.ErrForbidden).Once()

		result, err := svc.GetAll()

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

func TestSettingsService_Update(t *testing.T) {
	t.Run("forwards update to server", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Update", "key", "value").Return(nil).Once()
		require.NoError(t, svc.Update("key", "value"))
	})

	t.Run("propagates server authorization", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "executor")
		repo.On("Update", "key", "value").Return(models.ErrForbidden).Once()
		require.ErrorIs(t, svc.Update("key", "value"), models.ErrForbidden)
	})
}

func TestValidateRollbackMigrationRequest(t *testing.T) {
	valid := models.RollbackMigrationRequest{
		BackupCompleted:      true,
		BackupReference:      "smb://backup/docflow/2026-05-28_120000.tar",
		AcknowledgedDataLoss: true,
		Confirmation:         rollbackMigrationConfirmationPhrase,
	}

	tests := []struct {
		name    string
		req     models.RollbackMigrationRequest
		wantErr bool
	}{
		{name: "valid", req: valid, wantErr: false},
		{name: "backup not confirmed", req: models.RollbackMigrationRequest{
			BackupReference:      valid.BackupReference,
			AcknowledgedDataLoss: true,
			Confirmation:         valid.Confirmation,
		}, wantErr: true},
		{name: "empty backup reference", req: models.RollbackMigrationRequest{
			BackupCompleted:      true,
			AcknowledgedDataLoss: true,
			Confirmation:         valid.Confirmation,
		}, wantErr: true},
		{name: "data loss not acknowledged", req: models.RollbackMigrationRequest{
			BackupCompleted: true,
			BackupReference: valid.BackupReference,
			Confirmation:    valid.Confirmation,
		}, wantErr: true},
		{name: "wrong confirmation phrase", req: models.RollbackMigrationRequest{
			BackupCompleted:      true,
			BackupReference:      valid.BackupReference,
			AcknowledgedDataLoss: true,
			Confirmation:         "rollback",
		}, wantErr: true},
		{name: "trims confirmation phrase", req: models.RollbackMigrationRequest{
			BackupCompleted:      true,
			BackupReference:      "  " + valid.BackupReference + "  ",
			AcknowledgedDataLoss: true,
			Confirmation:         "  " + valid.Confirmation + "  ",
		}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRollbackMigrationRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				appErr, ok := models.AsAppError(err)
				require.True(t, ok)
				assert.Equal(t, "VALIDATION_ERROR", appErr.Kind)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMigrationCompatibilityAppError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "schema too new",
			err: &database.MigrationCompatibilityError{
				CurrentVersion:         8,
				LatestAvailableVersion: 7,
				SchemaTooNew:           true,
			},
			msg: "Версия схемы БД (8) новее миграций",
		},
		{
			name: "dirty schema",
			err: &database.MigrationCompatibilityError{
				CurrentVersion:         7,
				LatestAvailableVersion: 7,
				Dirty:                  true,
			},
			msg: "Миграция БД версии 7 завершилась с ошибкой",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := migrationCompatibilityAppError(tt.err)
			appErr, ok := models.AsAppError(err)
			require.True(t, ok)
			assert.Equal(t, "CONFLICT", appErr.Kind)
			assert.Equal(t, 409, appErr.Code)
			assert.Contains(t, appErr.Message, tt.msg)
		})
	}

	err := migrationCompatibilityAppError(assert.AnError)
	assert.ErrorIs(t, err, assert.AnError)
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

	t.Run("default on empty setting", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: " "}, nil).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(15*1024*1024), size)
	})

	t.Run("default on invalid setting", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "large"}, nil).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(15*1024*1024), size)
	})

	t.Run("default on out of range setting", func(t *testing.T) {
		svc, repo := setupSettingsService(t, "admin")
		repo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "2048"}, nil).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(DefaultAttachmentSizeMB*1024*1024), size)
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
	t.Run("forbidden non-admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "executor")
		err := svc.RunMigrations("Passw0rd!")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Недостаточно прав")
	})

	t.Run("calls server as current admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		client := &fakeServerMigrationClient{}
		svc.SetMigrationClient(client)

		require.NoError(t, svc.RunMigrations("Passw0rd!"))
		assert.Equal(t, 1, client.applyCalls)
		assert.Equal(t, "admin_set", client.login)
		assert.Equal(t, "Passw0rd!", client.password)
	})

	t.Run("requires password", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		err := svc.RunMigrations(" ")
		require.ErrorContains(t, err, "Введите пароль")
	})
}

func TestSettingsService_RunMigrationsReconcilesSchemaLifecycle(t *testing.T) {
	t.Run("successful migration", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		client := &fakeServerMigrationClient{}
		lifecycle := &fakeSchemaLifecycle{}
		svc.SetMigrationClient(client)
		ConfigureSchemaLifecycle(svc.authService, svc, lifecycle)

		require.NoError(t, svc.RunMigrations("Passw0rd!"))
		assert.Equal(t, 1, client.applyCalls)
		assert.Equal(t, 1, lifecycle.reconcileCalls)
	})

	t.Run("failed migration", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		client := &fakeServerMigrationClient{applyErr: assert.AnError}
		lifecycle := &fakeSchemaLifecycle{}
		svc.SetMigrationClient(client)
		ConfigureSchemaLifecycle(svc.authService, svc, lifecycle)

		require.Error(t, svc.RunMigrations("Passw0rd!"))
		assert.Equal(t, 1, client.applyCalls)
		assert.Zero(t, lifecycle.reconcileCalls)
	})
}

func TestSettingsService_GetMigrationStatus(t *testing.T) {
	t.Run("forbidden non-admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "clerk")
		status, err := svc.GetMigrationStatus()
		require.Error(t, err)
		require.Nil(t, status)
		assert.Contains(t, err.Error(), "Недостаточно прав")
	})

	t.Run("success admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		svc.SetMigrationClient(&fakeServerMigrationClient{status: database.MigrationStatus{CurrentVersion: 7, UpToDate: true}})
		status, err := svc.GetMigrationStatus()
		require.NoError(t, err)
		assert.EqualValues(t, 7, status.CurrentVersion)
	})
}

func TestSettingsService_RollbackMigration(t *testing.T) {
	// Откат последней примененной миграции базы данных
	validReq := models.RollbackMigrationRequest{
		BackupCompleted:      true,
		BackupReference:      "smb://backup/docflow/2026-05-28_120000.tar",
		AcknowledgedDataLoss: true,
		Confirmation:         rollbackMigrationConfirmationPhrase,
		Password:             "Passw0rd!",
	}

	t.Run("forbidden non-admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "executor")
		err := svc.RollbackMigration(validReq)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Недостаточно прав")
	})

	t.Run("admin requires rollback guardrails", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		err := svc.RollbackMigration(models.RollbackMigrationRequest{})
		require.Error(t, err)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "VALIDATION_ERROR", appErr.Kind)
	})

	t.Run("success admin", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		client := &fakeServerMigrationClient{}
		svc.SetMigrationClient(client)
		require.NoError(t, svc.RollbackMigration(validReq))
		assert.Equal(t, 1, client.rollbackCalls)
	})
}

func TestSettingsService_RollbackCoordinatesSchemaLifecycle(t *testing.T) {
	validReq := models.RollbackMigrationRequest{
		BackupCompleted:      true,
		BackupReference:      "smb://backup/docflow/2026-07-22_120000.tar",
		AcknowledgedDataLoss: true,
		Confirmation:         rollbackMigrationConfirmationPhrase,
		Password:             "Passw0rd!",
	}

	t.Run("successful rollback", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		client := &fakeServerMigrationClient{}
		lifecycle := &fakeSchemaLifecycle{}
		svc.SetMigrationClient(client)
		ConfigureSchemaLifecycle(svc.authService, svc, lifecycle)

		require.NoError(t, svc.RollbackMigration(validReq))
		assert.Equal(t, 1, client.rollbackCalls)
		assert.Equal(t, 1, lifecycle.reconcileCalls)
	})

	t.Run("failed rollback does not reconcile", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		client := &fakeServerMigrationClient{rollbackErr: assert.AnError}
		lifecycle := &fakeSchemaLifecycle{}
		svc.SetMigrationClient(client)
		ConfigureSchemaLifecycle(svc.authService, svc, lifecycle)

		require.Error(t, svc.RollbackMigration(validReq))
		assert.Equal(t, 1, client.rollbackCalls)
		assert.Zero(t, lifecycle.reconcileCalls)
	})
}

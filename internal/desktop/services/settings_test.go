package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

type fakeServerMigrationClient struct {
	status        dto.MigrationStatus
	statusErr     error
	applyErr      error
	rollbackErr   error
	applyCalls    int
	rollbackCalls int
	login         string
	password      string
}

type fakeServerSettingsClient struct {
	store ports.SettingsStore
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

func (c *fakeServerMigrationClient) Status(context.Context) (*dto.MigrationStatus, error) {
	if c.statusErr != nil {
		return nil, c.statusErr
	}
	return &c.status, nil
}

func (c *fakeServerMigrationClient) Apply(_ context.Context, login, password string) (*dto.MigrationStatus, error) {
	c.applyCalls++
	c.login, c.password = login, password
	if c.applyErr != nil {
		return nil, c.applyErr
	}
	return &c.status, nil
}

func (c *fakeServerMigrationClient) Rollback(_ context.Context, login, password string, _ models.RollbackMigrationRequest) (*dto.MigrationStatus, error) {
	c.rollbackCalls++
	c.login, c.password = login, password
	if c.rollbackErr != nil {
		return nil, c.rollbackErr
	}
	return &c.status, nil
}

type settingsTestPrincipal struct {
	admin bool
	login string
}

func (p *settingsTestPrincipal) GetCurrentUser() (*dto.User, error) {
	return &dto.User{Login: p.login}, nil
}
func (p *settingsTestPrincipal) RequireSystemPermissionWithoutSchemaCheck(string) error {
	if !p.admin {
		return models.ErrForbidden
	}
	return nil
}
func setupSettingsService(t *testing.T, role string) (*SettingsService, *mocks.SettingsStore) {
	t.Helper()
	store := mocks.NewSettingsStore(t)
	return NewSettingsService(&settingsTestPrincipal{admin: role == "admin", login: role + "_set"}, &fakeServerSettingsClient{store: store}, &fakeServerMigrationClient{}, nil), store
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
		svc.migrationClient = client

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
		svc.migrationClient = client
		svc.schemaLifecycle = lifecycle

		require.NoError(t, svc.RunMigrations("Passw0rd!"))
		assert.Equal(t, 1, client.applyCalls)
		assert.Equal(t, 1, lifecycle.reconcileCalls)
	})

	t.Run("failed migration", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		client := &fakeServerMigrationClient{applyErr: assert.AnError}
		lifecycle := &fakeSchemaLifecycle{}
		svc.migrationClient = client
		svc.schemaLifecycle = lifecycle

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
		svc.migrationClient = &fakeServerMigrationClient{status: dto.MigrationStatus{CurrentVersion: 7, UpToDate: true}}
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
		svc.migrationClient = client
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
		svc.migrationClient = client
		svc.schemaLifecycle = lifecycle

		require.NoError(t, svc.RollbackMigration(validReq))
		assert.Equal(t, 1, client.rollbackCalls)
		assert.Equal(t, 1, lifecycle.reconcileCalls)
	})

	t.Run("failed rollback does not reconcile", func(t *testing.T) {
		svc, _ := setupSettingsService(t, "admin")
		client := &fakeServerMigrationClient{rollbackErr: assert.AnError}
		lifecycle := &fakeSchemaLifecycle{}
		svc.migrationClient = client
		svc.schemaLifecycle = lifecycle

		require.Error(t, svc.RollbackMigration(validReq))
		assert.Equal(t, 1, client.rollbackCalls)
		assert.Zero(t, lifecycle.reconcileCalls)
	})
}

type fakeSchemaLifecycle struct{ reconcileCalls int }

func (l *fakeSchemaLifecycle) ReconcileSchema() { l.reconcileCalls++ }

func TestSettingsServiceMissingClients(t *testing.T) {
	svc := NewSettingsService(&settingsTestPrincipal{admin: true}, nil, nil, nil)
	_, err := svc.GetAll()
	require.ErrorContains(t, err, "не настроен")
	require.ErrorContains(t, svc.Update("key", "value"), "не настроен")
	_, err = svc.GetMigrationStatus()
	require.ErrorContains(t, err, "не настроен")
	require.ErrorContains(t, svc.RunMigrations("password"), "не настроен")
	require.ErrorContains(t, svc.RollbackMigration(models.RollbackMigrationRequest{
		BackupCompleted: true, BackupReference: "backup", AcknowledgedDataLoss: true,
		Confirmation: rollbackMigrationConfirmationPhrase, Password: "password",
	}), "не настроен")
	assert.Empty(t, svc.GetOrganizationShortName())
	assert.False(t, svc.IsAssignmentCompletionAttachmentsEnabled())
}

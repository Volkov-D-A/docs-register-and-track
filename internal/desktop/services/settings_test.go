package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
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
	settings  []models.SystemSetting
	setting   *models.SystemSetting
	listErr   error
	getErr    error
	updateErr error
	getKey    string
	updateKey string
	value     string
}

func (c *fakeServerSettingsClient) ListSettings(context.Context) ([]models.SystemSetting, error) {
	return c.settings, c.listErr
}

func (c *fakeServerSettingsClient) GetSystemSetting(_ context.Context, key string) (*models.SystemSetting, error) {
	c.getKey = key
	return c.setting, c.getErr
}

func (c *fakeServerSettingsClient) UpdateSystemSetting(_ context.Context, key, value string) error {
	c.updateKey, c.value = key, value
	return c.updateErr
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
func (p *settingsTestPrincipal) RequireSystemPermission(string) error {
	if !p.admin {
		return models.ErrForbidden
	}
	return nil
}
func setupSettingsService(t *testing.T, role string) (*SettingsService, *fakeServerSettingsClient) {
	t.Helper()
	client := &fakeServerSettingsClient{}
	return NewSettingsService(&settingsTestPrincipal{admin: role == "admin", login: role + "_set"}, client, &fakeServerMigrationClient{}), client
}

func TestSettingsService_GetAll(t *testing.T) {
	// Получение полного списка системных настроек от сервера
	t.Run("success", func(t *testing.T) {
		svc, client := setupSettingsService(t, "admin")
		settings := []models.SystemSetting{{Key: "k1", Value: "v1"}}
		client.settings = settings
		result, err := svc.GetAll()
		require.NoError(t, err)
		assert.Equal(t, settings, result)
	})

	t.Run("propagates server error", func(t *testing.T) {
		svc, client := setupSettingsService(t, "admin")
		client.listErr = models.ErrForbidden

		result, err := svc.GetAll()

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, result)
	})
}

func TestSettingsService_Update(t *testing.T) {
	t.Run("forwards update to server", func(t *testing.T) {
		svc, client := setupSettingsService(t, "admin")
		require.NoError(t, svc.Update("key", "value"))
		assert.Equal(t, "key", client.updateKey)
		assert.Equal(t, "value", client.value)
	})

	t.Run("propagates server authorization", func(t *testing.T) {
		svc, client := setupSettingsService(t, "executor")
		client.updateErr = models.ErrForbidden
		require.ErrorIs(t, svc.Update("key", "value"), models.ErrForbidden)
		assert.Equal(t, "key", client.updateKey)
		assert.Equal(t, "value", client.value)
	})
}

func TestValidateRollbackMigrationRequest(t *testing.T) {
	valid := models.RollbackMigrationRequest{
		BackupCompleted:      true,
		BackupReference:      "00000000-0000-4000-8000-000000000001",
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
		svc, client := setupSettingsService(t, "admin")
		client.setting = &models.SystemSetting{Key: "organization_short_name", Value: "Custom Short"}
		name := svc.GetOrganizationShortName()
		assert.Equal(t, "Custom Short", name)
		assert.Equal(t, "organization_short_name", client.getKey)
	})

	t.Run("default on error", func(t *testing.T) {
		svc, client := setupSettingsService(t, "admin")
		client.getErr = assert.AnError
		name := svc.GetOrganizationShortName()
		assert.Equal(t, "", name)
	})
}

func TestSettingsService_IsAssignmentCompletionAttachmentsEnabled(t *testing.T) {
	t.Run("from settings enabled", func(t *testing.T) {
		svc, client := setupSettingsService(t, "admin")
		client.setting = &models.SystemSetting{
			Key:   "assignment_completion_attachments_enabled",
			Value: "true",
		}
		assert.True(t, svc.IsAssignmentCompletionAttachmentsEnabled())
		assert.Equal(t, "assignment_completion_attachments_enabled", client.getKey)
	})

	t.Run("from settings disabled", func(t *testing.T) {
		svc, client := setupSettingsService(t, "admin")
		client.setting = &models.SystemSetting{
			Key:   "assignment_completion_attachments_enabled",
			Value: "false",
		}
		assert.False(t, svc.IsAssignmentCompletionAttachmentsEnabled())
	})

	t.Run("default on error", func(t *testing.T) {
		svc, client := setupSettingsService(t, "admin")
		client.getErr = assert.AnError
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
		BackupReference:      "00000000-0000-4000-8000-000000000001",
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

func TestSettingsServiceMissingClients(t *testing.T) {
	svc := NewSettingsService(&settingsTestPrincipal{admin: true}, nil, nil)
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

package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

func (s *SettingsService) StartBackupOperation(kind string, req models.BackupOperationRequest) (models.BackupOperationStarted, error) {
	c, err := s.backupClient()
	if err != nil {
		return models.BackupOperationStarted{}, err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.StartBackupOperation(ctx, kind, req)
}

func (s *SettingsService) GetBackupOperation(id, token string) (models.BackupOperation, error) {
	c, err := s.backupClient()
	if err != nil {
		return models.BackupOperation{}, err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.GetBackupOperation(ctx, id, token)
}

func (s *SettingsService) CancelBackupOperation(id string) error {
	c, err := s.backupClient()
	if err != nil {
		return err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.CancelBackupOperation(ctx, id)
}

func (s *SettingsService) backupClient() (serverclient.BackupClient, error) {
	client, ok := s.settingsClient.(serverclient.BackupClient)
	if !ok {
		return nil, models.NewConflict("Клиент резервирования недоступен")
	}
	return client, nil
}
func (s *SettingsService) GetBackupSettings() (models.BackupSettingsResponse, error) {
	c, err := s.backupClient()
	if err != nil {
		return models.BackupSettingsResponse{}, err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.GetBackupSettings(ctx)
}
func (s *SettingsService) SaveBackupSettings(value models.BackupSettingsUpdate) error {
	c, err := s.backupClient()
	if err != nil {
		return err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.SaveBackupSettings(ctx, value)
}
func (s *SettingsService) CheckBackupConnection() error {
	c, err := s.backupClient()
	if err != nil {
		return err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.CheckBackupConnection(ctx)
}
func (s *SettingsService) StartBackup() (models.BackupJob, error) {
	c, err := s.backupClient()
	if err != nil {
		return models.BackupJob{}, err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.StartBackup(ctx)
}
func (s *SettingsService) ListBackups() ([]models.BackupJob, error) {
	c, err := s.backupClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.ListBackups(ctx)
}

func (s *SettingsService) ListBackupCopies() ([]models.BackupCopy, error) {
	c, err := s.backupClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.ListBackupCopies(ctx)
}
func (s *SettingsService) CancelBackup(id string) error {
	c, err := s.backupClient()
	if err != nil {
		return err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.CancelBackup(ctx, id)
}

func (s *SettingsService) RetryBackup(id string) error {
	c, err := s.backupClient()
	if err != nil {
		return err
	}
	ctx, cancel := settingsContext()
	defer cancel()
	return c.RetryBackup(ctx, id)
}

package serverclient

import (
	"context"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"net/http"
	"net/url"
)

type BackupClient interface {
	RetryBackup(context.Context, string) error
	GetBackupSettings(context.Context) (models.BackupSettingsResponse, error)
	SaveBackupSettings(context.Context, models.BackupSettingsUpdate) error
	CheckBackupConnection(context.Context) error
	StartBackup(context.Context) (models.BackupJob, error)
	ListBackups(context.Context) ([]models.BackupJob, error)
	CancelBackup(context.Context, string) error
}

func (c *Client) GetBackupSettings(ctx context.Context) (models.BackupSettingsResponse, error) {
	var out models.BackupSettingsResponse
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/admin/backups/settings", nil, 200, &out)
	return out, err
}
func (c *Client) SaveBackupSettings(ctx context.Context, value models.BackupSettingsUpdate) error {
	return c.doUserRequest(ctx, http.MethodPut, "/api/v1/admin/backups/settings", value, 204, nil)
}
func (c *Client) CheckBackupConnection(ctx context.Context) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/admin/backups/check", nil, 204, nil)
}
func (c *Client) StartBackup(ctx context.Context) (models.BackupJob, error) {
	var out models.BackupJob
	err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/admin/backups", nil, 202, &out)
	return out, err
}
func (c *Client) ListBackups(ctx context.Context) ([]models.BackupJob, error) {
	var out []models.BackupJob
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/admin/backups", nil, 200, &out)
	return out, err
}
func (c *Client) CancelBackup(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/admin/backups/"+url.PathEscape(id)+"/cancel", nil, 204, nil)
}

func (c *Client) RetryBackup(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/admin/backups/"+url.PathEscape(id)+"/retry", nil, 204, nil)
}

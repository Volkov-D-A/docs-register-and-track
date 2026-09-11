package serverclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"io"
	"net/http"
	"net/url"
)

type BackupClient interface {
	StartBackupOperation(context.Context, string, models.BackupOperationRequest) (models.BackupOperationStarted, error)
	GetBackupOperation(context.Context, string, string) (models.BackupOperation, error)
	CancelBackupOperation(context.Context, string) error
	ListBackupCopies(context.Context) ([]models.BackupCopy, error)
	RetryBackup(context.Context, string) error
	GetBackupSettings(context.Context) (models.BackupSettingsResponse, error)
	SaveBackupSettings(context.Context, models.BackupSettingsUpdate) error
	CheckBackupConnection(context.Context) error
	StartBackup(context.Context) (models.BackupJob, error)
	ListBackups(context.Context) ([]models.BackupJob, error)
	CancelBackup(context.Context, string) error
}

func (c *Client) StartBackupOperation(ctx context.Context, kind string, req models.BackupOperationRequest) (models.BackupOperationStarted, error) {
	var out models.BackupOperationStarted
	if kind != "verify" && kind != "restore" && kind != "delete" {
		return out, fmt.Errorf("invalid backup operation")
	}
	err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/admin/backups/catalog/"+kind, req, 202, &out)
	if err == nil {
		c.observeOperation(out)
	}
	return out, err
}

// A status capability is never stored in the client's ordinary session state.
func (c *Client) GetBackupOperation(ctx context.Context, id, token string) (models.BackupOperation, error) {
	var out models.BackupOperation
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/admin/backups/operations/"+url.PathEscape(id), nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return out, decodeAuthError(resp)
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)
	return out, err
}

func (c *Client) CancelBackupOperation(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/admin/backups/operations/"+url.PathEscape(id)+"/cancel", nil, 204, nil)
}

func (c *Client) ListBackupCopies(ctx context.Context) ([]models.BackupCopy, error) {
	var out []models.BackupCopy
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/admin/backups/catalog", nil, 200, &out)
	return out, err
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

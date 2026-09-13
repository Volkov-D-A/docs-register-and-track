package serverclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type SettingsClient interface {
	ListSettings(context.Context) ([]models.SystemSetting, error)
	GetSystemSetting(context.Context, string) (*models.SystemSetting, error)
	UpdateSystemSetting(context.Context, string, string) error
}

type settingUpdateRequest struct {
	Value string `json:"value"`
}

func (c *Client) ListSettings(ctx context.Context) ([]models.SystemSetting, error) {
	var settings []models.SystemSetting
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/settings", nil, http.StatusOK, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (c *Client) GetSystemSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	path, err := settingPath(key)
	if err != nil {
		return nil, err
	}
	var setting models.SystemSetting
	if err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &setting); err != nil {
		return nil, err
	}
	return &setting, nil
}

func (c *Client) UpdateSystemSetting(ctx context.Context, key, value string) error {
	path, err := settingPath(key)
	if err != nil {
		return err
	}
	return c.doUserRequest(ctx, http.MethodPatch, path, settingUpdateRequest{Value: value}, http.StatusNoContent, nil)
}

func settingPath(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", models.NewBadRequest("ключ настройки обязателен")
	}
	return "/api/v1/settings/" + url.PathEscape(key), nil
}

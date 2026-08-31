package serverclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AuthClient interface {
	Login(context.Context, string, string) (*dto.User, error)
	Logout(context.Context) error
	Me(context.Context) (*dto.User, error)
	ChangePassword(context.Context, string, string) error
	ChangeRequiredPassword(context.Context, string, string, string) error
}

type loginResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	User        *dto.User `json:"user"`
}

func (c *Client) Login(ctx context.Context, login, password string) (*dto.User, error) {
	payload, err := json.Marshal(map[string]string{"login": login, "password": password})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/auth/login", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, decodeAuthError(resp)
	}
	var result loginResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode login response: %w", err)
	}
	if result.User == nil || result.AccessToken == "" {
		return nil, fmt.Errorf("docflow-server returned an incomplete login response")
	}
	c.tokenMu.Lock()
	c.token = result.AccessToken
	c.tokenMu.Unlock()
	return result.User, nil
}

func (c *Client) Logout(ctx context.Context) error {
	defer func() {
		c.tokenMu.Lock()
		c.token = ""
		c.tokenMu.Unlock()
	}()
	req, err := c.authenticatedRequest(ctx, http.MethodPost, "/api/v1/auth/logout")
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return decodeAuthError(resp)
	}
	return nil
}

func (c *Client) Me(ctx context.Context) (*dto.User, error) {
	req, err := c.authenticatedRequest(ctx, http.MethodGet, "/api/v1/auth/me")
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, decodeAuthError(resp)
	}
	var user dto.User
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode current user response: %w", err)
	}
	return &user, nil
}

func (c *Client) ChangePassword(ctx context.Context, oldPassword, newPassword string) error {
	payload, err := json.Marshal(map[string]string{"oldPassword": oldPassword, "newPassword": newPassword})
	if err != nil {
		return err
	}
	req, err := c.authenticatedRequestWithBody(ctx, http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		// The server may have committed the password change before the connection
		// failed, so the old session can no longer be trusted locally.
		c.clearToken()
		return fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return decodeAuthError(resp)
	}
	c.clearToken()
	return nil
}

func (c *Client) ChangeRequiredPassword(ctx context.Context, login, oldPassword, newPassword string) error {
	payload, err := json.Marshal(map[string]string{"login": login, "oldPassword": oldPassword, "newPassword": newPassword})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/auth/change-required-password", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return decodeAuthError(resp)
	}
	return nil
}

func (c *Client) authenticatedRequest(ctx context.Context, method, path string) (*http.Request, error) {
	return c.authenticatedRequestWithBody(ctx, method, path, nil)
}

func (c *Client) authenticatedRequestWithBody(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	c.tokenMu.RLock()
	token := c.token
	c.tokenMu.RUnlock()
	if token == "" {
		return nil, models.ErrUnauthorized
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

func (c *Client) clearToken() {
	c.tokenMu.Lock()
	c.token = ""
	c.tokenMu.Unlock()
}

func decodeAuthError(resp *http.Response) error {
	var body struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body)
	switch body.Code {
	case "invalid_credentials":
		return models.ErrInvalidCredentials
	case "user_locked":
		return models.ErrUserLocked
	case "user_inactive":
		return models.ErrUserNotActive
	case "password_change_required":
		return models.ErrPasswordChangeRequired
	case "wrong_password":
		return models.ErrWrongPassword
	case "invalid_request", "validation_error":
		return models.NewBadRequest(body.Error)
	case "password_change_not_required", "conflict":
		return models.NewConflict(body.Error)
	case "authentication_required", "session_invalid":
		return models.ErrUnauthorized
	case "forbidden":
		return models.ErrForbidden
	case "not_found":
		return models.NewNotFound(body.Error)
	default:
		if body.Error == "" {
			body.Error = resp.Status
		}
		return fmt.Errorf("docflow-server %s: %s", body.Code, body.Error)
	}
}

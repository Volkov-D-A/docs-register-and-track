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
	UpdateProfile(context.Context, models.UpdateProfileRequest) error
}

type InitialSetupClient interface {
	NeedsInitialSetup(context.Context) (bool, error)
	InitialSetup(context.Context, string) error
}

func (c *Client) NeedsInitialSetup(ctx context.Context) (bool, error) {
	var result struct {
		Required bool `json:"required"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/auth/setup-required", nil)
	if err != nil {
		return false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, decodeAuthError(resp)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil {
		return false, err
	}
	return result.Required, nil
}

func (c *Client) InitialSetup(ctx context.Context, password string) error {
	data, err := json.Marshal(map[string]string{"password": password})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/auth/setup", bytes.NewReader(data))
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

func (c *Client) UpdateProfile(ctx context.Context, input models.UpdateProfileRequest) error {
	return c.doUserRequest(ctx, http.MethodPatch, "/api/v1/profile", input, http.StatusNoContent, nil)
}

type loginResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	User        *dto.User `json:"user"`
}

func (c *Client) Login(ctx context.Context, login, password string) (*dto.User, error) {
	c.tokenMu.Lock()
	c.loginAttempt++
	attempt := c.loginAttempt
	c.tokenMu.Unlock()
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
	if c.loginAttempt != attempt {
		c.tokenMu.Unlock()
		return nil, models.ErrUnauthorized
	}
	oldCancel := c.sessionCancel
	c.sessionContext, c.sessionCancel = context.WithCancel(context.Background())
	c.token = result.AccessToken
	c.sessionUserID = result.User.ID
	c.sessionRevision++
	c.sessionReason = ""
	c.tokenMu.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
	return result.User, nil
}

func (c *Client) Logout(ctx context.Context) error {
	// Clear locally before the network call; a slow logout must not erase a new login.
	c.tokenMu.Lock()
	c.loginAttempt++
	token := c.token
	c.token = ""
	c.sessionUserID = ""
	c.sessionRevision++
	c.sessionReason = "logout"
	state, handler := c.sessionStateLocked(), c.onSessionEnded
	cancel := c.sessionCancel
	c.sessionContext, c.sessionCancel = nil, nil
	c.tokenMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if handler != nil {
		handler(state)
	}
	if token == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/auth/logout", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	// This request revokes the captured old token; its result cannot change local state.
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
	resp, err := c.doAuthenticated(req)
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
	resp, err := c.doAuthenticated(req)
	if err != nil {
		// The server may have committed the password change before the connection
		// failed, so the old session can no longer be trusted locally.
		c.endSession(sessionForRequest(req), "password_changed")
		return fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return decodeAuthError(resp)
	}
	c.endSession(sessionForRequest(req), "password_changed")
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
	revision := c.sessionRevision
	c.tokenMu.RUnlock()
	if token == "" {
		return nil, models.ErrUnauthorized
	}
	req, err := http.NewRequestWithContext(withRequestSession(ctx, revision, token), method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

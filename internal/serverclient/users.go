package serverclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type UserClient interface {
	ListUsers(context.Context) ([]dto.User, error)
	CreateUser(context.Context, models.CreateUserRequest) (*dto.User, error)
	UpdateUser(context.Context, models.UpdateUserRequest) (*dto.User, error)
	ResetUserPassword(context.Context, string) (string, error)
}

func (c *Client) ListUsers(ctx context.Context) ([]dto.User, error) {
	var users []dto.User
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/users", nil, http.StatusOK, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (c *Client) CreateUser(ctx context.Context, input models.CreateUserRequest) (*dto.User, error) {
	var user dto.User
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/users", input, http.StatusCreated, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) UpdateUser(ctx context.Context, input models.UpdateUserRequest) (*dto.User, error) {
	if input.ID == "" {
		return nil, models.NewBadRequest("идентификатор пользователя обязателен")
	}
	var user dto.User
	path := "/api/v1/users/" + url.PathEscape(input.ID)
	if err := c.doUserRequest(ctx, http.MethodPatch, path, input, http.StatusOK, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) ResetUserPassword(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", models.NewBadRequest("идентификатор пользователя обязателен")
	}
	var response struct {
		TemporaryPassword string `json:"temporaryPassword"`
	}
	path := "/api/v1/users/" + url.PathEscape(userID) + "/reset-password"
	if err := c.doUserRequest(ctx, http.MethodPost, path, nil, http.StatusOK, &response); err != nil {
		return "", err
	}
	if response.TemporaryPassword == "" {
		return "", fmt.Errorf("docflow-server returned an incomplete reset password response")
	}
	return response.TemporaryPassword, nil
}

func (c *Client) doUserRequest(ctx context.Context, method, path string, body any, expectedStatus int, result any) error {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode user request: %w", err)
		}
		payload = bytes.NewReader(data)
	}
	req, err := c.authenticatedRequestWithBody(ctx, method, path, payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		return decodeAuthError(resp)
	}
	if result == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result); err != nil {
		return fmt.Errorf("decode user response: %w", err)
	}
	return nil
}

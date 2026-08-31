package serverclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type UserAccessClient interface {
	GetUserAccessProfile(context.Context, string) (*models.UserDocumentAccessProfile, error)
	UpdateUserAccessProfile(context.Context, models.UpdateUserDocumentAccessRequest) error
}

type UserSubstitutionAdminClient interface {
	GetUserSubstitution(context.Context, string) (*dto.UserSubstitution, error)
	UpdateUserSubstitution(context.Context, models.UpdateUserSubstitutionRequest) (*dto.UserSubstitution, error)
}

type DepartmentLookupClient interface {
	ListDepartments(context.Context) ([]dto.Department, error)
}

type CurrentAccessClient interface {
	GetCurrentAccessSummary(context.Context) (*dto.CurrentAccessSummary, error)
}

func (c *Client) GetUserAccessProfile(ctx context.Context, userID string) (*models.UserDocumentAccessProfile, error) {
	if userID == "" {
		return nil, models.NewBadRequest("идентификатор пользователя обязателен")
	}
	var result models.UserDocumentAccessProfile
	path := "/api/v1/users/" + url.PathEscape(userID) + "/access-profile"
	if err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateUserAccessProfile(ctx context.Context, input models.UpdateUserDocumentAccessRequest) error {
	if input.UserID == "" {
		return models.NewBadRequest("идентификатор пользователя обязателен")
	}
	path := "/api/v1/users/" + url.PathEscape(input.UserID) + "/access-profile"
	return c.doUserRequest(ctx, http.MethodPut, path, input, http.StatusNoContent, nil)
}

func (c *Client) GetUserSubstitution(ctx context.Context, userID string) (*dto.UserSubstitution, error) {
	if userID == "" {
		return nil, models.NewBadRequest("идентификатор пользователя обязателен")
	}
	var result *dto.UserSubstitution
	path := "/api/v1/users/" + url.PathEscape(userID) + "/substitution"
	if err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateUserSubstitution(ctx context.Context, input models.UpdateUserSubstitutionRequest) (*dto.UserSubstitution, error) {
	if input.PrincipalUserID == "" {
		return nil, models.NewBadRequest("идентификатор пользователя обязателен")
	}
	var result *dto.UserSubstitution
	path := "/api/v1/users/" + url.PathEscape(input.PrincipalUserID) + "/substitution"
	if err := c.doUserRequest(ctx, http.MethodPut, path, input, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListDepartments(ctx context.Context) ([]dto.Department, error) {
	var result []dto.Department
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/departments", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetCurrentAccessSummary(ctx context.Context) (*dto.CurrentAccessSummary, error) {
	var result dto.CurrentAccessSummary
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/access/current", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

package serverclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type ReferenceClient interface {
	ListOrganizations(context.Context, string) ([]dto.Organization, error)
	ResolveOrganization(context.Context, string) (*dto.Organization, error)
	UpdateOrganization(context.Context, string, string) error
	DeleteOrganization(context.Context, string) error
	MergeOrganizations(context.Context, string, string) error
	ListResolutionExecutors(context.Context, string) ([]dto.ResolutionExecutor, error)
	ResolveResolutionExecutor(context.Context, string) (*dto.ResolutionExecutor, error)
	UpdateResolutionExecutor(context.Context, string, string) error
	DeleteResolutionExecutor(context.Context, string) error
}

type referenceNameRequest struct {
	Name string `json:"name"`
}

func (c *Client) ListOrganizations(ctx context.Context, query string) ([]dto.Organization, error) {
	var result []dto.Organization
	path := referenceListPath("/api/v1/references/organizations", query)
	if err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ResolveOrganization(ctx context.Context, name string) (*dto.Organization, error) {
	var result dto.Organization
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/references/organizations/resolve", referenceNameRequest{Name: name}, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateOrganization(ctx context.Context, id, name string) error {
	path, err := referenceItemPath("/api/v1/references/organizations/", id, "неверный ID записи справочника")
	if err != nil {
		return err
	}
	return c.doUserRequest(ctx, http.MethodPatch, path, referenceNameRequest{Name: name}, http.StatusNoContent, nil)
}

func (c *Client) DeleteOrganization(ctx context.Context, id string) error {
	path, err := referenceItemPath("/api/v1/references/organizations/", id, "неверный ID записи справочника")
	if err != nil {
		return err
	}
	return c.doUserRequest(ctx, http.MethodDelete, path, nil, http.StatusNoContent, nil)
}

func (c *Client) MergeOrganizations(ctx context.Context, sourceID, targetID string) error {
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID исходной организации", err)
	}
	targetUUID, err := uuid.Parse(targetID)
	if err != nil {
		return models.NewBadRequestWrapped("неверный ID целевой организации", err)
	}
	if sourceUUID == targetUUID {
		return models.NewBadRequest("нельзя объединить организацию саму с собой")
	}
	path := "/api/v1/references/organizations/" + url.PathEscape(sourceID)
	body := struct {
		TargetID string `json:"targetId"`
	}{TargetID: targetID}
	return c.doUserRequest(ctx, http.MethodPost, path+"/merge", body, http.StatusNoContent, nil)
}

func (c *Client) ListResolutionExecutors(ctx context.Context, query string) ([]dto.ResolutionExecutor, error) {
	var result []dto.ResolutionExecutor
	path := referenceListPath("/api/v1/references/resolution-executors", query)
	if err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ResolveResolutionExecutor(ctx context.Context, name string) (*dto.ResolutionExecutor, error) {
	var result dto.ResolutionExecutor
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/references/resolution-executors/resolve", referenceNameRequest{Name: name}, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateResolutionExecutor(ctx context.Context, id, name string) error {
	path, err := referenceItemPath("/api/v1/references/resolution-executors/", id, "неверный ID записи справочника")
	if err != nil {
		return err
	}
	return c.doUserRequest(ctx, http.MethodPatch, path, referenceNameRequest{Name: name}, http.StatusNoContent, nil)
}

func (c *Client) DeleteResolutionExecutor(ctx context.Context, id string) error {
	path, err := referenceItemPath("/api/v1/references/resolution-executors/", id, "неверный ID записи справочника")
	if err != nil {
		return err
	}
	return c.doUserRequest(ctx, http.MethodDelete, path, nil, http.StatusNoContent, nil)
}

func referenceListPath(path, query string) string {
	if query == "" {
		return path
	}
	values := url.Values{}
	values.Set("query", query)
	return path + "?" + values.Encode()
}

func referenceItemPath(prefix, id, message string) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", models.NewBadRequestWrapped(message, err)
	}
	return prefix + url.PathEscape(id), nil
}

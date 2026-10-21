package serverclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type DeadlineClient interface {
	PreviewDeadlineSeriesResume(context.Context, string) (*models.DeadlineSeriesResumePreview, error)
	ListDeadlineExecutors(context.Context, bool) ([]models.DeadlineExecutor, error)
	SaveExternalDeadlineExecutor(context.Context, string, models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error)
	ListDeadlineControls(context.Context, models.DeadlineControlFilter) (*models.DeadlineControlPage, error)
	CreateDeadlineControl(context.Context, models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error)
	UpdateDeadlineControl(context.Context, string, models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error)
	CompleteDeadlineControl(context.Context, string, models.CompleteDeadlineRequest) (*models.DocumentDeadlineControl, error)
	CancelDeadlineControl(context.Context, string, models.CancelDeadlineRequest) (*models.DocumentDeadlineControl, error)
	CreateDeadlineSeries(context.Context, models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error)
	GetDeadlineSeries(context.Context, string) (*models.DocumentDeadlineSeries, error)
	UpdateDeadlineSeries(context.Context, string, models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error)
	StopDeadlineSeries(context.Context, string, models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error)
	ResumeDeadlineSeries(context.Context, string, models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error)
}

func deadlineClientRequest[Result any](c *Client, ctx context.Context, method, path string, request any, status int) (*Result, error) {
	var result Result
	if err := c.doUserRequest(ctx, method, path, request, status, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) ListDeadlineExecutors(ctx context.Context, includeInactive bool) ([]models.DeadlineExecutor, error) {
	var result []models.DeadlineExecutor
	path := "/api/v1/deadline-executors"
	if includeInactive {
		path += "?includeInactive=true"
	}
	err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &result)
	return result, err
}
func (c *Client) SaveExternalDeadlineExecutor(ctx context.Context, id string, req models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error) {
	method, path, status := http.MethodPost, "/api/v1/deadline-executors", http.StatusCreated
	if id != "" {
		method, path, status = http.MethodPatch, path+"/"+url.PathEscape(id), http.StatusOK
	}
	return deadlineClientRequest[models.DeadlineExecutor](c, ctx, method, path, req, status)
}
func (c *Client) ListDeadlineControls(ctx context.Context, req models.DeadlineControlFilter) (*models.DeadlineControlPage, error) {
	return deadlineClientRequest[models.DeadlineControlPage](c, ctx, http.MethodPost, "/api/v1/deadline-controls/query", req, http.StatusOK)
}
func (c *Client) CreateDeadlineControl(ctx context.Context, req models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
	return deadlineClientRequest[models.DocumentDeadlineControl](c, ctx, http.MethodPost, "/api/v1/deadline-controls", req, http.StatusCreated)
}
func (c *Client) UpdateDeadlineControl(ctx context.Context, id string, req models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
	return deadlineClientRequest[models.DocumentDeadlineControl](c, ctx, http.MethodPatch, "/api/v1/deadline-controls/"+url.PathEscape(id), req, http.StatusOK)
}
func (c *Client) CompleteDeadlineControl(ctx context.Context, id string, req models.CompleteDeadlineRequest) (*models.DocumentDeadlineControl, error) {
	return deadlineClientRequest[models.DocumentDeadlineControl](c, ctx, http.MethodPost, "/api/v1/deadline-controls/"+url.PathEscape(id)+"/complete", req, http.StatusOK)
}
func (c *Client) CancelDeadlineControl(ctx context.Context, id string, req models.CancelDeadlineRequest) (*models.DocumentDeadlineControl, error) {
	return deadlineClientRequest[models.DocumentDeadlineControl](c, ctx, http.MethodPost, "/api/v1/deadline-controls/"+url.PathEscape(id)+"/cancel", req, http.StatusOK)
}
func (c *Client) CreateDeadlineSeries(ctx context.Context, req models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error) {
	return deadlineClientRequest[models.DocumentDeadlineSeries](c, ctx, http.MethodPost, "/api/v1/deadline-series", req, http.StatusCreated)
}
func (c *Client) GetDeadlineSeries(ctx context.Context, id string) (*models.DocumentDeadlineSeries, error) {
	return deadlineClientRequest[models.DocumentDeadlineSeries](c, ctx, http.MethodGet, "/api/v1/deadline-series/"+url.PathEscape(id), nil, http.StatusOK)
}
func (c *Client) UpdateDeadlineSeries(ctx context.Context, id string, req models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error) {
	return deadlineClientRequest[models.DocumentDeadlineSeries](c, ctx, http.MethodPatch, "/api/v1/deadline-series/"+url.PathEscape(id), req, http.StatusOK)
}
func (c *Client) StopDeadlineSeries(ctx context.Context, id string, req models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error) {
	return deadlineClientRequest[models.DocumentDeadlineSeries](c, ctx, http.MethodPost, "/api/v1/deadline-series/"+url.PathEscape(id)+"/stop", req, http.StatusOK)
}
func (c *Client) ResumeDeadlineSeries(ctx context.Context, id string, req models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error) {
	return deadlineClientRequest[models.DocumentDeadlineSeries](c, ctx, http.MethodPost, "/api/v1/deadline-series/"+url.PathEscape(id)+"/resume", req, http.StatusOK)
}

func (c *Client) PreviewDeadlineSeriesResume(ctx context.Context, id string) (*models.DeadlineSeriesResumePreview, error) {
	return deadlineClientRequest[models.DeadlineSeriesResumePreview](c, ctx, http.MethodGet, "/api/v1/deadline-series/"+url.PathEscape(id)+"/resume-preview", nil, http.StatusOK)
}

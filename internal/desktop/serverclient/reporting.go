package serverclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type ReportingClient interface {
	RunReport(context.Context, models.ReportRequest) (*models.ReportResult, error)
	GetReportFilters(context.Context) (*models.ReportFilters, error)
	ExportReport(context.Context, models.ReportRequest, string) (string, io.ReadCloser, error)
	CreateReportSchedule(context.Context, models.ReportScheduleRequest) (*models.ReportSchedule, error)
	ListReportSchedules(context.Context) ([]models.ReportSchedule, error)
	DisableReportSchedule(context.Context, string) error
	ListReportRuns(context.Context, string) ([]models.ReportRun, error)
	DownloadReportRun(context.Context, string) (string, io.ReadCloser, error)
	RetryReportRun(context.Context, string) error
}

func (c *Client) RunReport(ctx context.Context, req models.ReportRequest) (*models.ReportResult, error) {
	var result models.ReportResult
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/reports/run", req, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetReportFilters(ctx context.Context) (*models.ReportFilters, error) {
	var result models.ReportFilters
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/reports/filters", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ExportReport(ctx context.Context, req models.ReportRequest, format string) (string, io.ReadCloser, error) {
	if format != "xlsx" && format != "pdf" {
		return "", nil, models.NewBadRequest("неизвестный формат отчёта")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", nil, err
	}
	httpReq, err := c.authenticatedRequestWithBody(ctx, http.MethodPost, "/api/v1/reports/export/"+format, bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.doAuthenticated(httpReq)
	if err != nil {
		return "", nil, fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return "", nil, decodeAuthError(resp)
	}
	const maxReportBytes = 50 << 20
	if resp.ContentLength < 0 || resp.ContentLength > maxReportBytes {
		resp.Body.Close()
		return "", nil, fmt.Errorf("invalid report size")
	}
	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil || params["filename"] == "" {
		resp.Body.Close()
		return "", nil, fmt.Errorf("missing report filename")
	}
	return params["filename"], resp.Body, nil
}

func (c *Client) CreateReportSchedule(ctx context.Context, req models.ReportScheduleRequest) (*models.ReportSchedule, error) {
	var result models.ReportSchedule
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/reports/schedules", req, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListReportSchedules(ctx context.Context) ([]models.ReportSchedule, error) {
	var result []models.ReportSchedule
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/reports/schedules", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DisableReportSchedule(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodDelete, "/api/v1/reports/schedules/"+id, nil, http.StatusNoContent, nil)
}

func (c *Client) ListReportRuns(ctx context.Context, id string) ([]models.ReportRun, error) {
	var result []models.ReportRun
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/reports/schedules/"+id+"/runs", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DownloadReportRun(ctx context.Context, id string) (string, io.ReadCloser, error) {
	req, err := c.authenticatedRequest(ctx, http.MethodGet, "/api/v1/reports/runs/"+id+"/file")
	if err != nil {
		return "", nil, err
	}
	resp, err := c.doAuthenticated(req)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return "", nil, decodeAuthError(resp)
	}
	if resp.ContentLength < 0 || resp.ContentLength > 50<<20 {
		resp.Body.Close()
		return "", nil, fmt.Errorf("invalid report size")
	}
	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil || params["filename"] == "" {
		resp.Body.Close()
		return "", nil, fmt.Errorf("missing report filename")
	}
	return params["filename"], resp.Body, nil
}

func (c *Client) RetryReportRun(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/reports/runs/"+id+"/retry", nil, http.StatusNoContent, nil)
}

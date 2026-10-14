package serverclient

import (
	"context"
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type WorkspaceClient interface {
	GetWorkspaceDocuments(context.Context) (*dto.WorkspaceDocuments, error)
}

type StatisticsClient interface {
	GetDocumentStatistics(context.Context) (*models.DocumentStatistics, error)
	GetDocumentReport(context.Context, string, string, string, string, string, string) (*models.DocumentStatisticsReport, error)
	GetDocumentFilterOptions(context.Context) (*models.DocumentStatisticsFilters, error)
	GetDeadlineStatistics(context.Context) (*models.DeadlineStatistics, error)
	GetDeadlineReport(context.Context, string, string, bool, string) (*models.DeadlineStatisticsReport, error)
	GetDeadlineFilterOptions(context.Context) (*models.DeadlineStatisticsFilters, error)
	GetSystemStatistics(context.Context) (*models.SystemStatistics, error)
}

type documentStatisticsReportRequest struct {
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	GroupBy        string `json:"groupBy"`
	KindCode       string `json:"kindCode"`
	NomenclatureID string `json:"nomenclatureId"`
	UserID         string `json:"userId"`
}

type deadlineStatisticsReportRequest struct {
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	OnlyOverdue bool   `json:"onlyOverdue"`
	UserID      string `json:"userId"`
}

func getStatistics[T any](ctx context.Context, c *Client, path string) (*T, error) {
	var result T
	if err := c.doUserRequest(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetDocumentStatistics(ctx context.Context) (*models.DocumentStatistics, error) {
	return getStatistics[models.DocumentStatistics](ctx, c, "/api/v1/statistics/documents")
}

func (c *Client) GetDocumentReport(ctx context.Context, startDate, endDate, groupBy, kindCode, nomenclatureID, userID string) (*models.DocumentStatisticsReport, error) {
	var result models.DocumentStatisticsReport
	req := documentStatisticsReportRequest{StartDate: startDate, EndDate: endDate, GroupBy: groupBy, KindCode: kindCode, NomenclatureID: nomenclatureID, UserID: userID}
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/statistics/documents/report", req, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetDocumentFilterOptions(ctx context.Context) (*models.DocumentStatisticsFilters, error) {
	return getStatistics[models.DocumentStatisticsFilters](ctx, c, "/api/v1/statistics/documents/filters")
}

func (c *Client) GetDeadlineStatistics(ctx context.Context) (*models.DeadlineStatistics, error) {
	return getStatistics[models.DeadlineStatistics](ctx, c, "/api/v1/statistics/controls")
}

func (c *Client) GetDeadlineReport(ctx context.Context, startDate, endDate string, onlyOverdue bool, userID string) (*models.DeadlineStatisticsReport, error) {
	var result models.DeadlineStatisticsReport
	req := deadlineStatisticsReportRequest{StartDate: startDate, EndDate: endDate, OnlyOverdue: onlyOverdue, UserID: userID}
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/statistics/controls/report", req, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetDeadlineFilterOptions(ctx context.Context) (*models.DeadlineStatisticsFilters, error) {
	return getStatistics[models.DeadlineStatisticsFilters](ctx, c, "/api/v1/statistics/controls/filters")
}

func (c *Client) GetSystemStatistics(ctx context.Context) (*models.SystemStatistics, error) {
	return getStatistics[models.SystemStatistics](ctx, c, "/api/v1/statistics/system")
}

func (c *Client) GetWorkspaceDocuments(ctx context.Context) (*dto.WorkspaceDocuments, error) {
	return getStatistics[dto.WorkspaceDocuments](ctx, c, "/api/v1/workspace/documents")
}

package serverclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type WorkspaceClient interface {
	GetWorkspaceOverview(context.Context, string, string) (*dto.WorkspaceOverview, error)
	ListWorkspaceAcknowledgments(context.Context, string, int, int) (*dto.PagedResult[dto.WorkspaceAcknowledgment], error)
}

type StatisticsClient interface {
	GetDocumentStatistics(context.Context) (*models.DocumentStatistics, error)
	GetDocumentReport(context.Context, string, string, string, string, string, string) (*models.DocumentStatisticsReport, error)
	GetDocumentFilterOptions(context.Context) (*models.DocumentStatisticsFilters, error)
	GetAssignmentStatistics(context.Context) (*models.AssignmentStatistics, error)
	GetAssignmentReport(context.Context, string, string, bool, string) (*models.AssignmentStatisticsReport, error)
	GetAssignmentFilterOptions(context.Context) (*models.AssignmentStatisticsFilters, error)
	GetSystemStatistics(context.Context) (*models.SystemStatistics, error)
	GetStorageStatisticsStatus(context.Context) (*models.StorageStatisticsStatus, error)
	RetryStorageStatisticsRefresh(context.Context) (*models.StorageStatisticsStatus, error)
}

type documentStatisticsReportRequest struct {
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
	GroupBy        string `json:"groupBy"`
	KindCode       string `json:"kindCode"`
	NomenclatureID string `json:"nomenclatureId"`
	UserID         string `json:"userId"`
}

type assignmentStatisticsReportRequest struct {
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

func (c *Client) GetWorkspaceOverview(ctx context.Context, assignmentMode, acknowledgmentMode string) (*dto.WorkspaceOverview, error) {
	query := url.Values{}
	query.Set("assignmentMode", assignmentMode)
	query.Set("acknowledgmentMode", acknowledgmentMode)
	return getStatistics[dto.WorkspaceOverview](ctx, c, "/api/v1/workspace/overview?"+query.Encode())
}
func (c *Client) ListWorkspaceAcknowledgments(ctx context.Context, mode string, page, pageSize int) (*dto.PagedResult[dto.WorkspaceAcknowledgment], error) {
	query := url.Values{}
	query.Set("mode", mode)
	query.Set("page", strconv.Itoa(page))
	query.Set("pageSize", strconv.Itoa(pageSize))
	return getStatistics[dto.PagedResult[dto.WorkspaceAcknowledgment]](ctx, c, "/api/v1/workspace/acknowledgments?"+query.Encode())
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

func (c *Client) GetAssignmentStatistics(ctx context.Context) (*models.AssignmentStatistics, error) {
	return getStatistics[models.AssignmentStatistics](ctx, c, "/api/v1/statistics/assignments")
}

func (c *Client) GetAssignmentReport(ctx context.Context, startDate, endDate string, onlyOverdue bool, userID string) (*models.AssignmentStatisticsReport, error) {
	var result models.AssignmentStatisticsReport
	req := assignmentStatisticsReportRequest{StartDate: startDate, EndDate: endDate, OnlyOverdue: onlyOverdue, UserID: userID}
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/statistics/assignments/report", req, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetAssignmentFilterOptions(ctx context.Context) (*models.AssignmentStatisticsFilters, error) {
	return getStatistics[models.AssignmentStatisticsFilters](ctx, c, "/api/v1/statistics/assignments/filters")
}

func (c *Client) GetSystemStatistics(ctx context.Context) (*models.SystemStatistics, error) {
	return getStatistics[models.SystemStatistics](ctx, c, "/api/v1/statistics/system")
}

func (c *Client) GetStorageStatisticsStatus(ctx context.Context) (*models.StorageStatisticsStatus, error) {
	return getStatistics[models.StorageStatisticsStatus](ctx, c, "/api/v1/statistics/system/storage")
}

func (c *Client) RetryStorageStatisticsRefresh(ctx context.Context) (*models.StorageStatisticsStatus, error) {
	var result models.StorageStatisticsStatus
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/statistics/system/storage/retry", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

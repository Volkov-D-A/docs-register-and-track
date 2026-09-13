package serverclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AdminAuditClient interface {
	GetAdminAuditLog(context.Context, int, int) (*dto.AdminAuditLogPage, error)
}

type OutboxAdminClient interface {
	GetOutboxStats(context.Context) (models.OutboxStats, error)
	GetFailedOutboxEvents(context.Context, int) ([]models.FailedOutboxEvent, error)
	RequeueOutboxEvent(context.Context, string) error
}

func (c *Client) GetAdminAuditLog(ctx context.Context, page, pageSize int) (*dto.AdminAuditLogPage, error) {
	query := url.Values{"page": {strconv.Itoa(page)}, "pageSize": {strconv.Itoa(pageSize)}}
	var result dto.AdminAuditLogPage
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/admin/audit?"+query.Encode(), nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetOutboxStats(ctx context.Context) (models.OutboxStats, error) {
	var result models.OutboxStats
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/admin/outbox/stats", nil, http.StatusOK, &result)
	return result, err
}

func (c *Client) GetFailedOutboxEvents(ctx context.Context, limit int) ([]models.FailedOutboxEvent, error) {
	var result []models.FailedOutboxEvent
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/admin/outbox/failed?limit="+strconv.Itoa(limit), nil, http.StatusOK, &result)
	return result, err
}

func (c *Client) RequeueOutboxEvent(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/admin/outbox/"+url.PathEscape(id)+"/requeue", nil, http.StatusNoContent, nil)
}

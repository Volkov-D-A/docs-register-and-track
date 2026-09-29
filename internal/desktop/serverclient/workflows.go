package serverclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type UserEventClient interface {
	ListUserEvents(context.Context, models.UserEventFilter) (*dto.PagedResult[dto.UserEvent], error)
	GetUnreadUserEventCount(context.Context) (int, error)
	MarkUserEventRead(context.Context, string) error
	MarkDocumentUserEventsRead(context.Context, string) error
	MarkAllUserEventsRead(context.Context) error
}

type AdministrativeOrderAcknowledgmentClient interface {
	MarkAdministrativeOrderAcknowledged(context.Context, string) (*dto.AdministrativeOrderAcknowledgmentPerson, error)
}

func (c *Client) ListUserEvents(ctx context.Context, filter models.UserEventFilter) (*dto.PagedResult[dto.UserEvent], error) {
	var result dto.PagedResult[dto.UserEvent]
	err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/user-events/query", filter, http.StatusOK, &result)
	return &result, err
}

func (c *Client) GetUnreadUserEventCount(ctx context.Context) (int, error) {
	var result struct {
		Count int `json:"count"`
	}
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/user-events/unread-count", nil, http.StatusOK, &result)
	return result.Count, err
}

func (c *Client) MarkUserEventRead(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/user-events/"+url.PathEscape(id)+"/read", nil, http.StatusNoContent, nil)
}

func (c *Client) MarkDocumentUserEventsRead(ctx context.Context, documentID string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/user-events/documents/"+url.PathEscape(documentID)+"/read", nil, http.StatusNoContent, nil)
}

func (c *Client) MarkAllUserEventsRead(ctx context.Context) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/user-events/read-all", nil, http.StatusNoContent, nil)
}

func (c *Client) MarkAdministrativeOrderAcknowledged(ctx context.Context, personID string) (*dto.AdministrativeOrderAcknowledgmentPerson, error) {
	var result dto.AdministrativeOrderAcknowledgmentPerson
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/administrative-order-acknowledgments/"+url.PathEscape(personID)+"/confirm", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

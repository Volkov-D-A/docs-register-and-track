package serverclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AcknowledgmentClient interface {
	CreateAcknowledgment(context.Context, string, string, []string) (*dto.Acknowledgment, error)
	ListAcknowledgments(context.Context, string) ([]dto.Acknowledgment, error)
	ListPendingAcknowledgments(context.Context) ([]dto.Acknowledgment, error)
	ListPendingAcknowledgmentsByDocument(context.Context, string) ([]dto.Acknowledgment, error)
	ListActiveAcknowledgments(context.Context) ([]dto.Acknowledgment, error)
	MarkAcknowledgmentViewed(context.Context, string) error
	MarkAcknowledgmentConfirmed(context.Context, string) error
	DeleteAcknowledgment(context.Context, string) error
}

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

type createAcknowledgmentRequest struct {
	DocumentID string   `json:"documentId"`
	Content    string   `json:"content"`
	UserIDs    []string `json:"userIds"`
}

func (c *Client) CreateAcknowledgment(ctx context.Context, documentID, content string, userIDs []string) (*dto.Acknowledgment, error) {
	var result dto.Acknowledgment
	err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/acknowledgments", createAcknowledgmentRequest{DocumentID: documentID, Content: content, UserIDs: userIDs}, http.StatusCreated, &result)
	return &result, err
}

func (c *Client) ListAcknowledgments(ctx context.Context, documentID string) ([]dto.Acknowledgment, error) {
	var result []dto.Acknowledgment
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/acknowledgments?documentId="+url.QueryEscape(documentID), nil, http.StatusOK, &result)
	return result, err
}

func (c *Client) ListPendingAcknowledgments(ctx context.Context) ([]dto.Acknowledgment, error) {
	var result []dto.Acknowledgment
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/acknowledgments/pending", nil, http.StatusOK, &result)
	return result, err
}

func (c *Client) ListPendingAcknowledgmentsByDocument(ctx context.Context, documentID string) ([]dto.Acknowledgment, error) {
	var result []dto.Acknowledgment
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/acknowledgments/pending/"+url.PathEscape(documentID), nil, http.StatusOK, &result)
	return result, err
}

func (c *Client) ListActiveAcknowledgments(ctx context.Context) ([]dto.Acknowledgment, error) {
	var result []dto.Acknowledgment
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/acknowledgments/active", nil, http.StatusOK, &result)
	return result, err
}

func (c *Client) MarkAcknowledgmentViewed(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/acknowledgments/"+url.PathEscape(id)+"/view", nil, http.StatusNoContent, nil)
}

func (c *Client) MarkAcknowledgmentConfirmed(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/acknowledgments/"+url.PathEscape(id)+"/confirm", nil, http.StatusNoContent, nil)
}

func (c *Client) DeleteAcknowledgment(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodDelete, "/api/v1/acknowledgments/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
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

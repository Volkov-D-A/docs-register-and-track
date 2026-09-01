package serverclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type LinkClient interface {
	LinkDocuments(context.Context, string, string, string) (*dto.DocumentLink, error)
	UnlinkDocument(context.Context, string) error
	GetDocumentLinks(context.Context, string) ([]dto.DocumentLink, error)
	GetDocumentFlow(context.Context, string) (*models.GraphData, error)
}

type JournalClient interface {
	GetDocumentJournal(context.Context, string) ([]dto.JournalEntry, error)
}

type linkDocumentsRequest struct {
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
	LinkType string `json:"linkType"`
}

func (c *Client) LinkDocuments(ctx context.Context, sourceID, targetID, linkType string) (*dto.DocumentLink, error) {
	var result dto.DocumentLink
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/document-links", linkDocumentsRequest{SourceID: sourceID, TargetID: targetID, LinkType: linkType}, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UnlinkDocument(ctx context.Context, id string) error {
	return c.doUserRequest(ctx, http.MethodDelete, "/api/v1/document-links/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

func (c *Client) GetDocumentLinks(ctx context.Context, documentID string) ([]dto.DocumentLink, error) {
	var result []dto.DocumentLink
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/documents/"+url.PathEscape(documentID)+"/links", nil, http.StatusOK, &result)
	return result, err
}

func (c *Client) GetDocumentFlow(ctx context.Context, documentID string) (*models.GraphData, error) {
	var result models.GraphData
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/documents/"+url.PathEscape(documentID)+"/link-graph", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetDocumentJournal(ctx context.Context, documentID string) ([]dto.JournalEntry, error) {
	var result []dto.JournalEntry
	err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/documents/"+url.PathEscape(documentID)+"/journal", nil, http.StatusOK, &result)
	return result, err
}

package serverclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type DocumentQueryClient interface {
	GetDocumentCard(context.Context, string) (*dto.DocumentCard, error)
	ListDocuments(context.Context, string, models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error)
}

type documentListRequest struct {
	KindCode string                `json:"kindCode"`
	Filter   models.DocumentFilter `json:"filter"`
}

func (c *Client) GetDocumentCard(ctx context.Context, id string) (*dto.DocumentCard, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, models.NewBadRequestWrapped("неверный ID документа", err)
	}
	var card dto.DocumentCard
	if err := c.doUserRequest(ctx, http.MethodGet, "/api/v1/documents/"+url.PathEscape(id), nil, http.StatusOK, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

func (c *Client) ListDocuments(ctx context.Context, kindCode string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	kindCode = strings.TrimSpace(kindCode)
	if kindCode == "" {
		return nil, models.NewBadRequest("вид документа обязателен")
	}
	// AccessScope is deliberately excluded from JSON and is always resolved by
	// the server from the bearer principal.
	filter.AccessScope = nil
	var result dto.PagedResult[dto.DocumentListItem]
	request := documentListRequest{KindCode: kindCode, Filter: filter}
	if err := c.doUserRequest(ctx, http.MethodPost, "/api/v1/documents/query", request, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

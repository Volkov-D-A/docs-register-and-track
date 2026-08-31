package serverclient

import (
	"context"
	"encoding/json"
	"fmt"
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

type DocumentCommandClient interface {
	RegisterDocument(context.Context, string, any) (any, error)
	UpdateDocument(context.Context, string, any) (any, error)
	CreateAdminDocumentDraft(context.Context, string, any) (any, error)
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

func (c *Client) RegisterDocument(ctx context.Context, kindCode string, request any) (any, error) {
	key, err := commandIdempotencyKey(request)
	if err != nil {
		return nil, err
	}
	return c.doDocumentCommand(ctx, http.MethodPost, "/api/v1/documents/"+url.PathEscape(kindCode), request, http.StatusCreated, key, kindCode)
}

func (c *Client) UpdateDocument(ctx context.Context, kindCode string, request any) (any, error) {
	id, err := documentCommandStringField(request, "id")
	if err != nil {
		return nil, err
	}
	key := uuid.NewString()
	return c.doDocumentCommand(ctx, http.MethodPatch, "/api/v1/documents/"+url.PathEscape(kindCode)+"/"+url.PathEscape(id), request, http.StatusOK, key, kindCode)
}

func (c *Client) CreateAdminDocumentDraft(ctx context.Context, kindCode string, request any) (any, error) {
	return c.doDocumentCommand(ctx, http.MethodPost, "/api/v1/documents/"+url.PathEscape(kindCode)+"/admin-drafts", request, http.StatusCreated, uuid.NewString(), kindCode)
}

func commandIdempotencyKey(request any) (string, error) {
	value, err := documentCommandStringField(request, "idempotencyKey")
	if err != nil {
		return "", err
	}
	key, err := uuid.Parse(value)
	if err != nil || key == uuid.Nil {
		return "", models.NewBadRequest("неверный ключ идемпотентности")
	}
	return key.String(), nil
}

func documentCommandStringField(request any, field string) (string, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encode document command: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return "", fmt.Errorf("inspect document command: %w", err)
	}
	value, _ := values[field].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return "", models.NewBadRequest("неверные поля команды документа")
	}
	return value, nil
}

func (c *Client) doDocumentCommand(ctx context.Context, method, path string, request any, status int, key, kindCode string) (any, error) {
	var result any
	switch models.DocumentKind(kindCode) {
	case models.DocumentKindIncomingLetter:
		result = &dto.IncomingDocument{}
	case models.DocumentKindOutgoingLetter:
		result = &dto.OutgoingDocument{}
	case models.DocumentKindCitizenAppeal:
		result = &dto.CitizenAppealDocument{}
	case models.DocumentKindAdministrativeOrder:
		result = &dto.AdministrativeOrderDocument{}
	default:
		return nil, models.NewBadRequest("неподдерживаемый вид документа")
	}
	if err := c.doUserRequestWithIdempotency(ctx, method, path, request, status, result, key); err != nil {
		return nil, err
	}
	return result, nil
}

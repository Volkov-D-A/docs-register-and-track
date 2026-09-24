package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

var errServerDocumentQueryClientNotConfigured = errors.New("docflow-server document query client is not configured")

// DocumentQueryService is the Wails adapter for server-owned document queries.
type DocumentQueryService struct {
	server serverclient.DocumentQueryClient
}

func NewDocumentQueryService(client serverclient.DocumentQueryClient) *DocumentQueryService {
	return &DocumentQueryService{server: client}
}

func (s *DocumentQueryService) GetByID(id string) (*dto.DocumentCard, error) {
	if s.server == nil {
		return nil, errServerDocumentQueryClientNotConfigured
	}
	ctx, cancel := documentQueryContext()
	defer cancel()
	return s.server.GetDocumentCard(ctx, id)
}

func (s *DocumentQueryService) GetList(kindCode string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	if s.server == nil {
		return nil, errServerDocumentQueryClientNotConfigured
	}
	ctx, cancel := documentQueryContext()
	defer cancel()
	return s.server.ListDocuments(ctx, kindCode, filter)
}

func documentQueryContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

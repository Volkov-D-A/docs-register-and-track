package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

var errServerDocumentQueryClientNotConfigured = errors.New("docflow-server document query client is not configured")

// DocumentQueryService is the Wails adapter for server-owned document queries.
type DocumentQueryService struct {
	server  serverclient.DocumentQueryClient
	metrics *observability.Registry
}

func NewDocumentQueryService(client serverclient.DocumentQueryClient, metrics *observability.Registry) *DocumentQueryService {
	return &DocumentQueryService{server: client, metrics: metrics}
}

func (s *DocumentQueryService) GetByID(id string) (*dto.DocumentCard, error) {
	return operations.Measure(s.metrics, "documents.get_card", func() (*dto.DocumentCard, error) {
		if s.server == nil {
			return nil, errServerDocumentQueryClientNotConfigured
		}
		ctx, cancel := documentQueryContext()
		defer cancel()
		return s.server.GetDocumentCard(ctx, id)
	})
}

func (s *DocumentQueryService) GetList(kindCode string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	return operations.Measure(s.metrics, "documents.get_list", func() (*dto.PagedResult[dto.DocumentListItem], error) {
		if s.server == nil {
			return nil, errServerDocumentQueryClientNotConfigured
		}
		ctx, cancel := documentQueryContext()
		defer cancel()
		result, err := s.server.ListDocuments(ctx, kindCode, filter)
		if err == nil && result != nil && s.metrics != nil {
			s.metrics.AddCounter("documents.list.items", float64(len(result.Items)))
		}
		return result, err
	})
}

func documentQueryContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

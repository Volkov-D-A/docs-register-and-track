package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

var errServerDocumentQueryClientNotConfigured = errors.New("docflow-server document query client is not configured")

// DocumentQueryService is the Wails adapter for server-owned document queries.
type DocumentQueryService struct {
	server  serverclient.DocumentQueryClient
	metrics *observability.Registry
}

func NewDocumentQueryService() *DocumentQueryService {
	return &DocumentQueryService{}
}

func (s *DocumentQueryService) SetServerClient(client serverclient.DocumentQueryClient) {
	s.server = client
}

func (s *DocumentQueryService) SetOperationMetrics(metrics *observability.Registry) {
	s.metrics = metrics
}

func (s *DocumentQueryService) GetByID(id string) (*dto.DocumentCard, error) {
	return measureOperation(s.metrics, "documents.get_card", func() (*dto.DocumentCard, error) {
		if s.server == nil {
			return nil, errServerDocumentQueryClientNotConfigured
		}
		ctx, cancel := documentQueryContext()
		defer cancel()
		return s.server.GetDocumentCard(ctx, id)
	})
}

func (s *DocumentQueryService) GetList(kindCode string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	return measureOperation(s.metrics, "documents.get_list", func() (*dto.PagedResult[dto.DocumentListItem], error) {
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

// DocumentQueryEngine executes server-side document queries and access checks.
type DocumentQueryEngine struct {
	registry *DocumentKindQueryRegistry
	access   *DocumentAccessService
	metrics  *observability.Registry
}

func (s *DocumentQueryEngine) SetOperationMetrics(metrics *observability.Registry) {
	s.metrics = metrics
}

func NewDocumentQueryEngine(
	registry *DocumentKindQueryRegistry,
	access *DocumentAccessService,
) *DocumentQueryEngine {
	return &DocumentQueryEngine{
		registry: registry,
		access:   access,
	}
}

// GetByID возвращает общую карточку документа по его ID.
func (s *DocumentQueryEngine) GetByID(id string) (*dto.DocumentCard, error) {
	return measureOperation(s.metrics, "documents.get_card", func() (*dto.DocumentCard, error) {
		if err := s.access.RequireDomainRead(); err != nil {
			return nil, err
		}

		uid, err := uuid.Parse(id)
		if err != nil {
			return nil, models.NewBadRequestWrapped("неверный ID документа", err)
		}

		doc, err := s.access.RequireExists(uid)
		if err != nil {
			return nil, err
		}
		if err := s.access.RequireReadResolved(doc); err != nil {
			return nil, err
		}

		handler, err := s.registry.Get(doc.Kind)
		if err != nil {
			return nil, models.ErrForbidden
		}

		return handler.GetCard(uid)
	})
}

// GetList возвращает общий список документов указанного вида.
func (s *DocumentQueryEngine) GetList(kindCode string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	return measureOperation(s.metrics, "documents.get_list", func() (*dto.PagedResult[dto.DocumentListItem], error) {
		if err := s.access.RequireDomainRead(); err != nil {
			return nil, err
		}

		kind := models.DocumentKind(kindCode)
		scope, err := s.access.ResolveReadScope(kind)
		if err != nil {
			return nil, err
		}
		filter.AccessScope = scope

		handler, err := s.registry.Get(kind)
		if err != nil {
			return nil, models.ErrForbidden
		}

		result, err := handler.GetList(filter)
		if err == nil && result != nil && s.metrics != nil {
			s.metrics.AddCounter("documents.list.items", float64(len(result.Items)))
		}
		return result, err
	})
}

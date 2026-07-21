package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
)

// DocumentQueryService предоставляет общий read-only API для карточек документов.
type DocumentQueryService struct {
	registry *DocumentKindQueryRegistry
	access   *DocumentAccessService
	metrics  *observability.Registry
}

func (s *DocumentQueryService) SetOperationMetrics(metrics *observability.Registry) {
	s.metrics = metrics
}

// NewDocumentQueryService создает новый экземпляр DocumentQueryService.
func NewDocumentQueryService(
	registry *DocumentKindQueryRegistry,
	access *DocumentAccessService,
) *DocumentQueryService {
	return &DocumentQueryService{
		registry: registry,
		access:   access,
	}
}

// GetByID возвращает общую карточку документа по его ID.
func (s *DocumentQueryService) GetByID(id string) (*dto.DocumentCard, error) {
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
func (s *DocumentQueryService) GetList(kindCode string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	return measureOperation(s.metrics, "documents.get_list", func() (*dto.PagedResult[dto.DocumentListItem], error) {
		if err := s.access.RequireDomainRead(); err != nil {
			return nil, err
		}

		kind := models.DocumentKind(kindCode)
		scope, err := s.access.ResolveReadScope(kind)
		if err != nil {
			return nil, err
		}
		if scope.Restricted {
			filter.AllowedNomenclatureIDs = scope.AllowedNomenclatureIDs
			filter.AccessibleByUserID = scope.AccessibleByUserID
			filter.AccessibleByUserIDs = scope.AccessibleByUserIDs
		}
		// Pass an explicit scope even for full access. Repositories can therefore
		// distinguish an intentional unrestricted query from legacy direct calls.
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

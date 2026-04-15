package services

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentQueryService предоставляет общий read-only API для карточек документов.
type DocumentQueryService struct {
	registry *DocumentKindQueryRegistry
	access   *DocumentAccessService
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
	if err := s.access.RequireDomainRead(); err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
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
}

// GetList возвращает общий список документов указанного вида.
func (s *DocumentQueryService) GetList(kindCode string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
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
	}

	handler, err := s.registry.Get(kind)
	if err != nil {
		return nil, models.ErrForbidden
	}

	return handler.GetList(filter)
}

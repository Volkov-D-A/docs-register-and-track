package services

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

// DocumentQueryEngine executes server-side document queries and access checks.
type DocumentQueryEngine struct {
	registry *DocumentKindQueryRegistry
	access   *DocumentAccessService
	search   ports.DocumentSearchStore
}

func NewDocumentQueryEngine(
	registry *DocumentKindQueryRegistry,
	access *DocumentAccessService,
	search ports.DocumentSearchStore,
) *DocumentQueryEngine {
	return &DocumentQueryEngine{
		registry: registry,
		access:   access,
		search:   search,
	}
}

// GetByID возвращает общую карточку документа по его ID.
func (s *DocumentQueryEngine) GetByID(id string) (*dto.DocumentCard, error) {
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
}

// GetList возвращает общий список документов указанного вида.
func (s *DocumentQueryEngine) GetList(kindCode string, filter models.DocumentFilter) (*dto.CursorResult[dto.DocumentListItem], error) {
	if err := s.access.RequireDomainRead(); err != nil {
		return nil, err
	}

	kind := models.DocumentKind(kindCode)
	scope, err := s.access.ResolveReadScope()
	if err != nil {
		return nil, err
	}
	filter.AccessScope = scope

	handler, err := s.registry.Get(kind)
	if err != nil {
		return nil, models.ErrForbidden
	}

	return handler.GetList(filter)
}

// Search applies request-local read scopes before ranking and pagination.
func (s *DocumentQueryEngine) Search(request dto.DocumentSearchRequest) (*dto.DocumentSearchResult, error) {
	if err := s.access.RequireDomainRead(); err != nil {
		return nil, err
	}
	request.Query = strings.Join(strings.Fields(request.Query), " ")
	if request.Query == "" || utf8.RuneCountInString(request.Query) > 500 || len(strings.Fields(request.Query)) > 32 {
		return nil, models.NewBadRequest("Введите от 1 до 32 слов, не более 500 символов")
	}
	if request.Page <= 0 {
		request.Page = 1
	}
	if request.Page > 10000 {
		return nil, models.NewBadRequest("Слишком большой номер страницы")
	}
	if request.PageSize <= 0 {
		request.PageSize = 20
	}
	if request.PageSize > 100 {
		request.PageSize = 100
	}
	scopes := make(map[models.DocumentKind]models.DocumentAccessScope)
	var scope *models.DocumentAccessScope
	for _, spec := range models.AllDocumentKindSpecs() {
		if _, err := s.registry.Get(spec.Code); err != nil {
			continue
		}
		if scope == nil {
			var err error
			scope, err = s.access.ResolveReadScope()
			if err != nil {
				return nil, err
			}
		}
		scopes[spec.Code] = *scope
	}
	return s.search.SearchDocuments(request, scopes)
}

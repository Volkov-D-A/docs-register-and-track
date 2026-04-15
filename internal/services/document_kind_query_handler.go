package services

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentKindQueryHandler описывает read-only обработчик конкретного вида документа.
type DocumentKindQueryHandler interface {
	Kind() models.DocumentKind
	GetCard(id uuid.UUID) (*dto.DocumentCard, error)
	GetList(filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error)
}

// DocumentKindQueryRegistry хранит обработчики query-операций по видам документов.
type DocumentKindQueryRegistry struct {
	handlers map[models.DocumentKind]DocumentKindQueryHandler
}

// NewDocumentKindQueryRegistry создает registry обработчиков документов.
func NewDocumentKindQueryRegistry(handlers ...DocumentKindQueryHandler) *DocumentKindQueryRegistry {
	registry := &DocumentKindQueryRegistry{
		handlers: make(map[models.DocumentKind]DocumentKindQueryHandler, len(handlers)),
	}

	for _, handler := range handlers {
		registry.handlers[handler.Kind()] = handler
	}

	return registry
}

// Get возвращает обработчик query-операций по виду документа.
func (r *DocumentKindQueryRegistry) Get(kind models.DocumentKind) (DocumentKindQueryHandler, error) {
	handler, ok := r.handlers[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported document kind: %s", kind)
	}

	return handler, nil
}

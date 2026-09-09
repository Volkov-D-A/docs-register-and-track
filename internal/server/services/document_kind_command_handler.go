package services

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
)

func documentCommandHash(req any) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("encode document command hash: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

// DocumentKindCommandHandler описывает write-обработчик конкретного вида документа.
type DocumentKindCommandHandler interface {
	Kind() models.DocumentKind
	RegisterDocument(req any) (any, error)
	UpdateDocument(req any) (any, error)
}

type AdminDraftCommandHandler interface {
	CreateAdminDraft(req dto.AdminDraftCreateRequest) (any, error)
}

// DocumentKindCommandRegistry хранит обработчики command-операций по видам документов.
type DocumentKindCommandRegistry struct {
	handlers map[models.DocumentKind]DocumentKindCommandHandler
}

// NewDocumentKindCommandRegistry создает registry command-обработчиков документов.
func NewDocumentKindCommandRegistry(handlers ...DocumentKindCommandHandler) *DocumentKindCommandRegistry {
	registry := &DocumentKindCommandRegistry{
		handlers: make(map[models.DocumentKind]DocumentKindCommandHandler, len(handlers)),
	}

	for _, handler := range handlers {
		registry.handlers[handler.Kind()] = handler
	}

	return registry
}

// Get возвращает обработчик command-операций по виду документа.
func (r *DocumentKindCommandRegistry) Get(kind models.DocumentKind) (DocumentKindCommandHandler, error) {
	handler, ok := r.handlers[kind]
	if !ok {
		return nil, models.NewBadRequest("неподдерживаемый вид документа")
	}

	return handler, nil
}

// DocumentCommandEngine предоставляет общий command API для регистрации и обновления документов.
type DocumentCommandEngine struct {
	registry *DocumentKindCommandRegistry
	metrics  *observability.Registry
}

func NewDocumentCommandEngine(registry *DocumentKindCommandRegistry, metrics *observability.Registry) *DocumentCommandEngine {
	return &DocumentCommandEngine{registry: registry, metrics: metrics}
}

// Register делегирует регистрацию документа обработчику по kindCode.
func (s *DocumentCommandEngine) Register(kindCode string, req any) (any, error) {
	return operations.Measure(s.metrics, "documents.register", func() (any, error) {
		kind := models.DocumentKind(kindCode)
		handler, err := s.registry.Get(kind)
		if err != nil {
			return nil, models.ErrForbidden
		}

		normalizedReq, err := dto.NormalizeDocumentRegisterRequest(kind, req)
		if err != nil {
			return nil, err
		}

		result, err := handler.RegisterDocument(normalizedReq)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

// Update делегирует обновление документа обработчику по kindCode.
func (s *DocumentCommandEngine) Update(kindCode string, req any) (any, error) {
	return operations.Measure(s.metrics, "documents.update", func() (any, error) {
		kind := models.DocumentKind(kindCode)
		handler, err := s.registry.Get(kind)
		if err != nil {
			return nil, models.ErrForbidden
		}

		normalizedReq, err := dto.NormalizeDocumentUpdateRequest(kind, req)
		if err != nil {
			return nil, err
		}

		result, err := handler.UpdateDocument(normalizedReq)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

// CreateAdminDraft создает административный черновик с зарезервированным номером.
func (s *DocumentCommandEngine) CreateAdminDraft(kindCode string, req dto.AdminDraftCreateRequest) (any, error) {
	return operations.Measure(s.metrics, "documents.create_admin_draft", func() (any, error) {
		kind := models.DocumentKind(kindCode)
		handler, err := s.registry.Get(kind)
		if err != nil {
			return nil, models.ErrForbidden
		}
		draftHandler, ok := handler.(AdminDraftCommandHandler)
		if !ok {
			return nil, models.ErrForbidden
		}

		result, err := draftHandler.CreateAdminDraft(req)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
)

type DocumentCommandClient interface {
	RegisterDocument(context.Context, string, any) (any, error)
	UpdateDocument(context.Context, string, any) (any, error)
	CreateAdminDocumentDraft(context.Context, string, dto.AdminDraftCreateRequest) (any, error)
}

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

// DocumentRegistrationService предоставляет общий command API для регистрации и обновления документов.
type DocumentRegistrationService struct {
	registry  *DocumentKindCommandRegistry
	client    DocumentCommandClient
	lifecycle *operations.Lifecycle
	metrics   *observability.Registry
}

func NewDocumentRegistrationServiceWithClient(client DocumentCommandClient) *DocumentRegistrationService {
	return &DocumentRegistrationService{client: client}
}

// NewDocumentRegistrationService создает новый экземпляр DocumentRegistrationService.
func NewDocumentRegistrationService(registry *DocumentKindCommandRegistry) *DocumentRegistrationService {
	return &DocumentRegistrationService{registry: registry}
}

func (s *DocumentRegistrationService) SetOperationLifecycle(lifecycle *operations.Lifecycle) {
	s.lifecycle = lifecycle
}

func (s *DocumentRegistrationService) SetOperationMetrics(metrics *observability.Registry) {
	s.metrics = metrics
}

// Register делегирует регистрацию документа обработчику по kindCode.
func (s *DocumentRegistrationService) Register(kindCode string, req any) (any, error) {
	return operations.Measure(s.metrics, "documents.register", func() (any, error) {
		ctx, release := s.lifecycle.OperationContext()
		defer release()
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		kind := models.DocumentKind(kindCode)
		if s.client != nil {
			normalizedReq, err := dto.NormalizeDocumentRegisterRequest(kind, req)
			if err != nil {
				return nil, err
			}
			return s.client.RegisterDocument(ctx, kindCode, normalizedReq)
		}
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
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// Update делегирует обновление документа обработчику по kindCode.
func (s *DocumentRegistrationService) Update(kindCode string, req any) (any, error) {
	return operations.Measure(s.metrics, "documents.update", func() (any, error) {
		ctx, release := s.lifecycle.OperationContext()
		defer release()
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		kind := models.DocumentKind(kindCode)
		if s.client != nil {
			normalizedReq, err := dto.NormalizeDocumentUpdateRequest(kind, req)
			if err != nil {
				return nil, err
			}
			return s.client.UpdateDocument(ctx, kindCode, normalizedReq)
		}
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
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// CreateAdminDraft создает административный черновик с зарезервированным номером.
func (s *DocumentRegistrationService) CreateAdminDraft(kindCode string, req dto.AdminDraftCreateRequest) (any, error) {
	return operations.Measure(s.metrics, "documents.create_admin_draft", func() (any, error) {
		ctx, release := s.lifecycle.OperationContext()
		defer release()
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		kind := models.DocumentKind(kindCode)
		if s.client != nil {
			return s.client.CreateAdminDocumentDraft(ctx, kindCode, req)
		}
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
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return result, nil
	})
}

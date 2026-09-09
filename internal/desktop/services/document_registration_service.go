package services

import (
	"context"
	"errors"

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

// DocumentRegistrationService exposes document commands through the HTTP API.
type DocumentRegistrationService struct {
	client    DocumentCommandClient
	lifecycle *operations.Lifecycle
	metrics   *observability.Registry
}

func NewDocumentRegistrationService(client DocumentCommandClient, lifecycle *operations.Lifecycle, metrics *observability.Registry) *DocumentRegistrationService {
	return &DocumentRegistrationService{client: client, lifecycle: lifecycle, metrics: metrics}
}

// Register делегирует регистрацию документа обработчику по kindCode.
func (s *DocumentRegistrationService) Register(kindCode string, req any) (any, error) {
	return operations.Measure(s.metrics, "documents.register", func() (any, error) {
		ctx, release := s.lifecycle.OperationContext()
		defer release()
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if s.client == nil {
			return nil, errors.New("docflow-server document command client is not configured")
		}
		kind := models.DocumentKind(kindCode)
		normalizedReq, err := dto.NormalizeDocumentRegisterRequest(kind, req)
		if err != nil {
			return nil, err
		}
		return s.client.RegisterDocument(ctx, kindCode, normalizedReq)
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

		if s.client == nil {
			return nil, errors.New("docflow-server document command client is not configured")
		}
		kind := models.DocumentKind(kindCode)
		normalizedReq, err := dto.NormalizeDocumentUpdateRequest(kind, req)
		if err != nil {
			return nil, err
		}
		return s.client.UpdateDocument(ctx, kindCode, normalizedReq)
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

		if s.client == nil {
			return nil, errors.New("docflow-server document command client is not configured")
		}
		return s.client.CreateAdminDocumentDraft(ctx, kindCode, req)
	})
}

package services

import (
	"encoding/json"
	"fmt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentKindCommandHandler описывает write-обработчик конкретного вида документа.
type DocumentKindCommandHandler interface {
	Kind() models.DocumentKind
	RegisterDocument(req any) (any, error)
	UpdateDocument(req any) (any, error)
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
		return nil, fmt.Errorf("unsupported document kind: %s", kind)
	}

	return handler, nil
}

// DocumentRegistrationService предоставляет общий command API для регистрации и обновления документов.
type DocumentRegistrationService struct {
	registry *DocumentKindCommandRegistry
}

// NewDocumentRegistrationService создает новый экземпляр DocumentRegistrationService.
func NewDocumentRegistrationService(registry *DocumentKindCommandRegistry) *DocumentRegistrationService {
	return &DocumentRegistrationService{registry: registry}
}

// Register делегирует регистрацию документа обработчику по kindCode.
func (s *DocumentRegistrationService) Register(kindCode string, req any) (any, error) {
	kind := models.DocumentKind(kindCode)
	handler, err := s.registry.Get(kind)
	if err != nil {
		return nil, models.ErrForbidden
	}

	normalizedReq, err := normalizeRegisterRequest(kind, req)
	if err != nil {
		return nil, err
	}

	return handler.RegisterDocument(normalizedReq)
}

// Update делегирует обновление документа обработчику по kindCode.
func (s *DocumentRegistrationService) Update(kindCode string, req any) (any, error) {
	kind := models.DocumentKind(kindCode)
	handler, err := s.registry.Get(kind)
	if err != nil {
		return nil, models.ErrForbidden
	}

	normalizedReq, err := normalizeUpdateRequest(kind, req)
	if err != nil {
		return nil, err
	}

	return handler.UpdateDocument(normalizedReq)
}

func normalizeRegisterRequest(kind models.DocumentKind, req any) (any, error) {
	switch kind {
	case models.DocumentKindIncomingLetter:
		if typedReq, ok := req.(IncomingLetterRegisterRequest); ok {
			return typedReq, nil
		}

		var typedReq IncomingLetterRegisterRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindOutgoingLetter:
		if typedReq, ok := req.(OutgoingLetterRegisterRequest); ok {
			return typedReq, nil
		}

		var typedReq OutgoingLetterRegisterRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	default:
		return nil, models.ErrForbidden
	}
}

func normalizeUpdateRequest(kind models.DocumentKind, req any) (any, error) {
	switch kind {
	case models.DocumentKindIncomingLetter:
		if typedReq, ok := req.(IncomingLetterUpdateRequest); ok {
			return typedReq, nil
		}

		var typedReq IncomingLetterUpdateRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindOutgoingLetter:
		if typedReq, ok := req.(OutgoingLetterUpdateRequest); ok {
			return typedReq, nil
		}

		var typedReq OutgoingLetterUpdateRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	default:
		return nil, models.ErrForbidden
	}
}

func decodeDocumentCommandRequest(src any, dst any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to encode document command request: %w", err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("failed to decode document command request: %w", err)
	}

	return nil
}

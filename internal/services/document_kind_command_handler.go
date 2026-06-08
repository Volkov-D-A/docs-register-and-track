package services

import (
	"bytes"
	"encoding/json"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentKindCommandHandler описывает write-обработчик конкретного вида документа.
type DocumentKindCommandHandler interface {
	Kind() models.DocumentKind
	RegisterDocument(req any) (any, error)
	UpdateDocument(req any) (any, error)
}

type AdminDraftCommandHandler interface {
	CreateAdminDraft(req AdminDraftCreateRequest) (any, error)
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
	lifecycle *OperationLifecycle
}

// NewDocumentRegistrationService создает новый экземпляр DocumentRegistrationService.
func NewDocumentRegistrationService(registry *DocumentKindCommandRegistry) *DocumentRegistrationService {
	return &DocumentRegistrationService{registry: registry}
}

func (s *DocumentRegistrationService) SetOperationLifecycle(lifecycle *OperationLifecycle) {
	s.lifecycle = lifecycle
}

// Register делегирует регистрацию документа обработчику по kindCode.
func (s *DocumentRegistrationService) Register(kindCode string, req any) (any, error) {
	ctx, release := serviceOperationContext(s.lifecycle)
	defer release()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	kind := models.DocumentKind(kindCode)
	handler, err := s.registry.Get(kind)
	if err != nil {
		return nil, models.ErrForbidden
	}

	normalizedReq, err := normalizeRegisterRequest(kind, req)
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
}

// Update делегирует обновление документа обработчику по kindCode.
func (s *DocumentRegistrationService) Update(kindCode string, req any) (any, error) {
	ctx, release := serviceOperationContext(s.lifecycle)
	defer release()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	kind := models.DocumentKind(kindCode)
	handler, err := s.registry.Get(kind)
	if err != nil {
		return nil, models.ErrForbidden
	}

	normalizedReq, err := normalizeUpdateRequest(kind, req)
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
}

// CreateAdminDraft создает административный черновик с зарезервированным номером.
func (s *DocumentRegistrationService) CreateAdminDraft(kindCode string, req AdminDraftCreateRequest) (any, error) {
	ctx, release := serviceOperationContext(s.lifecycle)
	defer release()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
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
	case models.DocumentKindCitizenAppeal:
		if typedReq, ok := req.(CitizenAppealRegisterRequest); ok {
			return typedReq, nil
		}

		var typedReq CitizenAppealRegisterRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindAdministrativeOrder:
		if typedReq, ok := req.(AdministrativeOrderRegisterRequest); ok {
			return typedReq, nil
		}

		var typedReq AdministrativeOrderRegisterRequest
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
	case models.DocumentKindCitizenAppeal:
		if typedReq, ok := req.(CitizenAppealUpdateRequest); ok {
			return typedReq, nil
		}

		var typedReq CitizenAppealUpdateRequest
		if err := decodeDocumentCommandRequest(req, &typedReq); err != nil {
			return nil, err
		}

		return typedReq, nil
	case models.DocumentKindAdministrativeOrder:
		if typedReq, ok := req.(AdministrativeOrderUpdateRequest); ok {
			return typedReq, nil
		}

		var typedReq AdministrativeOrderUpdateRequest
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
		return models.NewBadRequest("неверный формат команды документа")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return models.NewBadRequest("неверные поля команды документа")
	}

	return nil
}

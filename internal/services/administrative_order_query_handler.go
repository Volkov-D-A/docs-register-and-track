package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// AdministrativeOrderQueryHandler предоставляет read-only операции по приказам.
type AdministrativeOrderQueryHandler struct {
	repo AdministrativeOrderDocStore
}

// NewAdministrativeOrderQueryHandler создает query handler приказов.
func NewAdministrativeOrderQueryHandler(repo AdministrativeOrderDocStore) *AdministrativeOrderQueryHandler {
	return &AdministrativeOrderQueryHandler{repo: repo}
}

// Kind возвращает системный вид документа.
func (h *AdministrativeOrderQueryHandler) Kind() models.DocumentKind {
	return models.DocumentKindAdministrativeOrder
}

// GetCard возвращает карточку приказа.
func (h *AdministrativeOrderQueryHandler) GetCard(id uuid.UUID) (*dto.DocumentCard, error) {
	order, err := h.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return dto.MapAdministrativeOrderDocumentCard(order), nil
}

// GetList возвращает список приказов.
func (h *AdministrativeOrderQueryHandler) GetList(filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	result, err := h.repo.GetList(filter)
	if err != nil {
		return nil, err
	}
	return &dto.PagedResult[dto.DocumentListItem]{
		Items:      dto.MapDocumentListItemsFromAdministrativeOrders(result.Items),
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PageSize:   result.PageSize,
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
	}, nil
}

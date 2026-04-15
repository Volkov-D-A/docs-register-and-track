package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// IncomingLetterQueryHandler обслуживает read-only операции по входящим письмам.
type IncomingLetterQueryHandler struct {
	repo IncomingDocStore
}

// NewIncomingLetterQueryHandler создает обработчик входящих писем.
func NewIncomingLetterQueryHandler(repo IncomingDocStore) *IncomingLetterQueryHandler {
	return &IncomingLetterQueryHandler{repo: repo}
}

// Kind возвращает вид документа, который обслуживает handler.
func (h *IncomingLetterQueryHandler) Kind() models.DocumentKind {
	return models.DocumentKindIncomingLetter
}

// GetCard возвращает общую карточку входящего письма.
func (h *IncomingLetterQueryHandler) GetCard(id uuid.UUID) (*dto.DocumentCard, error) {
	incoming, err := h.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return dto.MapIncomingDocumentCard(incoming), nil
}

// GetList возвращает общий список входящих писем.
func (h *IncomingLetterQueryHandler) GetList(filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	res, err := h.repo.GetList(filter)
	if err != nil {
		return nil, err
	}

	return &dto.PagedResult[dto.DocumentListItem]{
		Items:      dto.MapDocumentListItemsFromIncoming(res.Items),
		TotalCount: res.TotalCount,
		Page:       res.Page,
		PageSize:   res.PageSize,
	}, nil
}

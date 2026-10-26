package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

// OutgoingLetterQueryHandler обслуживает read-only операции по исходящим письмам.
type OutgoingLetterQueryHandler struct {
	repo ports.OutgoingDocStore
}

// NewOutgoingLetterQueryHandler создает обработчик исходящих писем.
func NewOutgoingLetterQueryHandler(repo ports.OutgoingDocStore) *OutgoingLetterQueryHandler {
	return &OutgoingLetterQueryHandler{repo: repo}
}

// Kind возвращает вид документа, который обслуживает handler.
func (h *OutgoingLetterQueryHandler) Kind() models.DocumentKind {
	return models.DocumentKindOutgoingLetter
}

// GetCard возвращает общую карточку исходящего письма.
func (h *OutgoingLetterQueryHandler) GetCard(id uuid.UUID) (*dto.DocumentCard, error) {
	outgoing, err := h.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return dto.MapOutgoingDocumentCard(outgoing), nil
}

// GetList возвращает общий список исходящих писем.
func (h *OutgoingLetterQueryHandler) GetList(filter models.DocumentFilter) (*dto.CursorResult[dto.DocumentListItem], error) {
	outgoingFilter := models.OutgoingDocumentFilter{
		NomenclatureIDs: filter.NomenclatureIDs,
		AccessScope:     filter.AccessScope,
		DocumentTypeID:  filter.DocumentTypeID,
		OrgID:           filter.OrgID,
		DateFrom:        filter.DateFrom,
		DateTo:          filter.DateTo,
		Search:          filter.Search,
		OutgoingNumber:  filter.OutgoingNumber,
		RecipientName:   filter.RecipientName,
		PageSize:        filter.PageSize,
		Cursor:          filter.Cursor,
	}

	res, err := h.repo.GetList(outgoingFilter)
	if err != nil {
		return nil, err
	}

	return &dto.CursorResult[dto.DocumentListItem]{
		Items:      dto.MapDocumentListItemsFromOutgoing(res.Items),
		PageSize:   res.PageSize,
		NextCursor: res.NextCursor,
		HasMore:    res.HasMore,
	}, nil
}

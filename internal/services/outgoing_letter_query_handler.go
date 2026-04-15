package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// OutgoingLetterQueryHandler обслуживает read-only операции по исходящим письмам.
type OutgoingLetterQueryHandler struct {
	repo OutgoingDocStore
}

// NewOutgoingLetterQueryHandler создает обработчик исходящих писем.
func NewOutgoingLetterQueryHandler(repo OutgoingDocStore) *OutgoingLetterQueryHandler {
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
func (h *OutgoingLetterQueryHandler) GetList(filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	outgoingFilter := models.OutgoingDocumentFilter{
		NomenclatureIDs:        filter.NomenclatureIDs,
		AllowedNomenclatureIDs: filter.AllowedNomenclatureIDs,
		AccessibleByUserID:     filter.AccessibleByUserID,
		KindCode:               string(models.DocumentKindOutgoingLetter),
		DocumentTypeID:         filter.DocumentTypeID,
		OrgID:                  filter.OrgID,
		DateFrom:               filter.DateFrom,
		DateTo:                 filter.DateTo,
		Search:                 filter.Search,
		OutgoingNumber:         filter.OutgoingNumber,
		RecipientName:          filter.RecipientName,
		Page:                   filter.Page,
		PageSize:               filter.PageSize,
	}

	res, err := h.repo.GetList(outgoingFilter)
	if err != nil {
		return nil, err
	}

	return &dto.PagedResult[dto.DocumentListItem]{
		Items:      dto.MapDocumentListItemsFromOutgoing(res.Items),
		TotalCount: res.TotalCount,
		Page:       res.Page,
		PageSize:   res.PageSize,
	}, nil
}

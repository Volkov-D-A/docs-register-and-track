package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// CitizenAppealQueryHandler обслуживает read-only операции по обращениям граждан.
type CitizenAppealQueryHandler struct {
	repo CitizenAppealDocStore
}

// NewCitizenAppealQueryHandler создает обработчик обращений граждан.
func NewCitizenAppealQueryHandler(repo CitizenAppealDocStore) *CitizenAppealQueryHandler {
	return &CitizenAppealQueryHandler{repo: repo}
}

// Kind возвращает вид документа, который обслуживает handler.
func (h *CitizenAppealQueryHandler) Kind() models.DocumentKind {
	return models.DocumentKindCitizenAppeal
}

// GetCard возвращает общую карточку обращения граждан.
func (h *CitizenAppealQueryHandler) GetCard(id uuid.UUID) (*dto.DocumentCard, error) {
	doc, err := h.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return dto.MapCitizenAppealDocumentCard(doc), nil
}

// GetList возвращает общий список обращений граждан.
func (h *CitizenAppealQueryHandler) GetList(filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	res, err := h.repo.GetList(filter)
	if err != nil {
		return nil, err
	}

	return &dto.PagedResult[dto.DocumentListItem]{
		Items:      dto.MapDocumentListItemsFromCitizenAppeals(res.Items),
		TotalCount: res.TotalCount,
		Page:       res.Page,
		PageSize:   res.PageSize,
		NextCursor: res.NextCursor,
		HasMore:    res.HasMore,
	}, nil
}

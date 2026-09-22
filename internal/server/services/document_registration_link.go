package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func buildRegistrationLink(access *DocumentAccessService, kind models.DocumentKind, nomenclatureID uuid.UUID, req *dto.DocumentRegistrationLinkRequest) (*models.DocumentRegistrationLink, error) {
	if req == nil {
		return nil, nil
	}
	id, err := uuid.Parse(req.DocumentID)
	if err != nil || id == uuid.Nil {
		return nil, models.NewBadRequest("неверный ID связанного документа")
	}
	switch req.LinkType {
	case "reply", "follow_up", "related", "clarification", "order_amends", "order_cancels":
	default:
		return nil, models.NewBadRequest("неверный тип связи документов")
	}
	if err := access.RequireDocumentAction(id, "link"); err != nil {
		return nil, err
	}
	allowed, err := access.hasPermission(kind, "link")
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, models.ErrForbidden
	}
	// A new document has no assignments or acknowledgments yet. Resolve read
	// access from its kind/nomenclature before allocating its database ID.
	if err := access.RequireReadResolved(&models.Document{Kind: kind, NomenclatureID: nomenclatureID}); err != nil {
		return nil, err
	}
	if req.LinkType == "order_amends" || req.LinkType == "order_cancels" {
		existing, err := access.RequireExists(id)
		if err != nil {
			return nil, err
		}
		if err := validateDocumentLinkType(kind, existing.Kind, req.LinkType); err != nil {
			return nil, err
		}
	}
	return &models.DocumentRegistrationLink{DocumentID: id, LinkType: req.LinkType}, nil
}

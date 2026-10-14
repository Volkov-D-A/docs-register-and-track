package dto

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

func MapDocumentLink(m *models.DocumentLink) *DocumentLink {
	if m == nil {
		return nil
	}
	return &DocumentLink{ID: m.ID.String(), SourceKind: string(m.SourceKind), SourceID: m.SourceID.String(), TargetKind: string(m.TargetKind), LinkType: m.LinkType, SourceNumber: m.SourceNumber, TargetNumber: m.TargetNumber, SourceSubject: m.SourceSubject, TargetSubject: m.TargetSubject}
}

func MapDocumentLinks(m []models.DocumentLink) []DocumentLink {
	if m == nil {
		return nil
	}
	res := make([]DocumentLink, len(m))
	for i, item := range m {
		if mapped := MapDocumentLink(&item); mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}
func uuidString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

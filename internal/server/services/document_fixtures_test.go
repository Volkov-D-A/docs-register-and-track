package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

type kindBackedDocumentStore struct {
	incoming ports.IncomingDocStore
	outgoing ports.OutgoingDocStore
}

func (s *kindBackedDocumentStore) GetByID(id uuid.UUID) (*models.Document, error) {
	if s.incoming != nil {
		doc, err := s.incoming.GetByID(id)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			return &models.Document{ID: doc.ID, Kind: models.DocumentKindIncomingLetter}, nil
		}
	}
	if s.outgoing != nil {
		doc, err := s.outgoing.GetByID(id)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			return &models.Document{ID: doc.ID, Kind: models.DocumentKindOutgoingLetter}, nil
		}
	}
	return nil, nil
}

func (s *kindBackedDocumentStore) GetByIDs(ids []uuid.UUID) ([]models.Document, error) {
	result := make([]models.Document, 0, len(ids))
	for _, id := range ids {
		doc, err := s.GetByID(id)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			result = append(result, *doc)
		}
	}
	return result, nil
}

type roleMappedDocumentAccessStore struct {
	roles []string
}

func newRoleMappedDocumentAccessStore(roles ...string) ports.DocumentAccessStore {
	return &roleMappedDocumentAccessStore{roles: roles}
}

func (s *roleMappedDocumentAccessStore) HasPermission(kindCode, action string, departmentID, userID string) (bool, error) {
	for _, role := range s.roles {
		if role == "clerk" {
			switch action {
			case "register", "control_deadlines":
				return true, nil
			}
		}
	}
	return false, nil
}

// This fixture supplies visibility independently from management permissions.
// Clerk cases represent an explicit document-kind registration grant.
func (s *roleMappedDocumentAccessStore) CanReadDocument(_ uuid.UUID, _ uuid.UUID) (bool, error) {
	for _, role := range s.roles {
		if role == "clerk" {
			return true, nil
		}
	}
	return false, nil
}
func (s *roleMappedDocumentAccessStore) GetReadableDocumentIDs(user uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := map[uuid.UUID]bool{}
	for _, id := range ids {
		allowed, _ := s.CanReadDocument(user, id)
		if allowed {
			result[id] = true
		}
	}
	return result, nil
}

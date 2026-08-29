package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type kindBackedDocumentStore struct {
	incoming IncomingDocStore
	outgoing OutgoingDocStore
}

func (s *kindBackedDocumentStore) GetByID(id uuid.UUID) (*models.Document, error) {
	if s.incoming != nil {
		doc, err := s.incoming.GetByID(id)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			return &models.Document{ID: doc.ID, Kind: models.DocumentKindIncomingLetter, NomenclatureID: doc.NomenclatureID}, nil
		}
	}
	if s.outgoing != nil {
		doc, err := s.outgoing.GetByID(id)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			return &models.Document{ID: doc.ID, Kind: models.DocumentKindOutgoingLetter, NomenclatureID: doc.NomenclatureID}, nil
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

func newRoleMappedDocumentAccessStore(roles ...string) DocumentAccessStore {
	return &roleMappedDocumentAccessStore{roles: roles}
}

func (s *roleMappedDocumentAccessStore) HasPermission(kindCode, action string, departmentID, userID string) (bool, error) {
	for _, role := range s.roles {
		if role == "clerk" {
			switch action {
			case "create", "read", "update", "assign", "acknowledge", "upload", "link", "view_journal":
				return true, nil
			}
		}
	}
	for _, role := range s.roles {
		if role == "executor" {
			switch action {
			case "upload", "view_journal":
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *roleMappedDocumentAccessStore) HasSystemPermission(permission, userID string) (bool, error) {
	for _, role := range s.roles {
		if role == permission {
			return true, nil
		}
	}
	return false, nil
}

func (s *roleMappedDocumentAccessStore) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	return &models.UserDocumentAccessProfile{}, nil
}

func (s *roleMappedDocumentAccessStore) ReplaceUserAccessProfile(userID string, systemPermissions []models.UserSystemPermissionRule, permissions []models.UserDocumentPermissionRule) error {
	return nil
}

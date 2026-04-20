package services

import "github.com/Volkov-D-A/docs-register-and-track/internal/models"

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
			case "create", "read", "update", "delete", "assign", "acknowledge", "upload", "link", "view_journal":
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

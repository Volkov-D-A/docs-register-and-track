package repository

import (
	"fmt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentAccessRepository читает матрицу доступа document-domain.
type DocumentAccessRepository struct {
	db *database.DB
}

// NewDocumentAccessRepository создает новый экземпляр DocumentAccessRepository.
func NewDocumentAccessRepository(db *database.DB) *DocumentAccessRepository {
	return &DocumentAccessRepository{db: db}
}

// HasPermission проверяет, разрешено ли действие для вида документа по одной из subject-привязок пользователя.
func (r *DocumentAccessRepository) HasPermission(kindCode, action string, departmentID, userID string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM document_permissions p
			WHERE p.kind_code = $1
			  AND p.action = $2
			  AND p.is_allowed = true
			  AND (
				($3 <> '' AND p.subject_type = 'department' AND p.subject_key = $3)
				OR ($4 <> '' AND p.subject_type = 'user' AND p.subject_key = $4)
			  )
		)
	`, kindCode, action, departmentID, userID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("failed to check document permission: %w", err)
	}

	return allowed, nil
}

// HasSystemPermission проверяет наличие прямого системного права у пользователя.
func (r *DocumentAccessRepository) HasSystemPermission(permission, userID string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM user_system_permissions
			WHERE user_id = $1
			  AND permission = $2
			  AND is_allowed = true
		)
	`, userID, permission).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("failed to check system permission: %w", err)
	}
	return allowed, nil
}

// GetUserAccessProfile возвращает прямые document-domain права пользователя.
func (r *DocumentAccessRepository) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	systemRows, err := r.db.Query(`
		SELECT permission, is_allowed
		FROM user_system_permissions
		WHERE user_id = $1
		ORDER BY permission
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user system permissions: %w", err)
	}
	defer systemRows.Close()

	permissionsRows, err := r.db.Query(`
		SELECT kind_code, action, is_allowed
		FROM document_permissions
		WHERE subject_type = 'user' AND subject_key = $1
		ORDER BY kind_code, action
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user document permissions: %w", err)
	}
	defer permissionsRows.Close()

	profile := &models.UserDocumentAccessProfile{
		SystemPermissions: make([]models.UserSystemPermissionRule, 0),
		Permissions:       make([]models.UserDocumentPermissionRule, 0),
	}

	for systemRows.Next() {
		var item models.UserSystemPermissionRule
		if err := systemRows.Scan(&item.Permission, &item.IsAllowed); err != nil {
			return nil, err
		}
		profile.SystemPermissions = append(profile.SystemPermissions, item)
	}
	if err := systemRows.Err(); err != nil {
		return nil, err
	}

	for permissionsRows.Next() {
		var item models.UserDocumentPermissionRule
		if err := permissionsRows.Scan(&item.KindCode, &item.Action, &item.IsAllowed); err != nil {
			return nil, err
		}
		profile.Permissions = append(profile.Permissions, item)
	}
	if err := permissionsRows.Err(); err != nil {
		return nil, err
	}

	return profile, nil
}

// ReplaceUserAccessProfile заменяет прямые document-domain права пользователя.
func (r *DocumentAccessRepository) ReplaceUserAccessProfile(userID string, systemPermissions []models.UserSystemPermissionRule, permissions []models.UserDocumentPermissionRule) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin user access transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM user_system_permissions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("failed to clear user system permissions: %w", err)
	}

	for _, permission := range systemPermissions {
		if _, err := tx.Exec(`
			INSERT INTO user_system_permissions (user_id, permission, is_allowed)
			VALUES ($1, $2, $3)
		`, userID, permission.Permission, permission.IsAllowed); err != nil {
			return fmt.Errorf("failed to insert user system permission: %w", err)
		}
	}

	if _, err := tx.Exec(`DELETE FROM document_permissions WHERE subject_type = 'user' AND subject_key = $1`, userID); err != nil {
		return fmt.Errorf("failed to clear user document permissions: %w", err)
	}

	for _, permission := range permissions {
		if _, err := tx.Exec(`
			INSERT INTO document_permissions (kind_code, subject_type, subject_key, action, is_allowed)
			VALUES ($1, 'user', $2, $3, $4)
		`, permission.KindCode, userID, permission.Action, permission.IsAllowed); err != nil {
			return fmt.Errorf("failed to insert user document permission: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit user access transaction: %w", err)
	}

	return nil
}

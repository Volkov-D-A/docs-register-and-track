package repository

import (
	"fmt"

	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
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
func (r *DocumentAccessRepository) HasPermission(kindCode, action string, roles []string, departmentID, userID string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM document_permissions p
			WHERE p.kind_code = $1
			  AND p.action = $2
			  AND p.is_allowed = true
			  AND (
				(p.subject_type = 'role' AND p.subject_key = ANY($3))
				OR ($4 <> '' AND p.subject_type = 'department' AND p.subject_key = $4)
				OR ($5 <> '' AND p.subject_type = 'user' AND p.subject_key = $5)
			  )
		)
	`, kindCode, action, pq.Array(roles), departmentID, userID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("failed to check document permission: %w", err)
	}

	return allowed, nil
}

// GetVisibilityChannels возвращает каналы видимости, применимые к пользователю для вида документа.
func (r *DocumentAccessRepository) GetVisibilityChannels(kindCode string, roles []string, departmentID, userID string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT visibility_channel
		FROM document_visibility_rules
		WHERE kind_code = $1
		  AND (
			(subject_type = 'role' AND subject_key = ANY($2))
			OR ($3 <> '' AND subject_type = 'department' AND subject_key = $3)
			OR ($4 <> '' AND subject_type = 'user' AND subject_key = $4)
		  )
	`, kindCode, pq.Array(roles), departmentID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get visibility channels: %w", err)
	}
	defer rows.Close()

	channels := make([]string, 0)
	for rows.Next() {
		var channel string
		if err := rows.Scan(&channel); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

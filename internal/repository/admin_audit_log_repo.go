package repository

import (
	"database/sql"
	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/google/uuid"
)

// AdminAuditLogRepository предоставляет методы для работы с журналом действий администраторов.
type AdminAuditLogRepository struct {
	db *database.DB
}

// NewAdminAuditLogRepository создает новый экземпляр AdminAuditLogRepository.
func NewAdminAuditLogRepository(db *database.DB) *AdminAuditLogRepository {
	return &AdminAuditLogRepository{db: db}
}

// Create создает новую запись в журнале действий администраторов.
func (r *AdminAuditLogRepository) Create(req models.CreateAdminAuditLogRequest) (uuid.UUID, error) {
	return r.create(req, "")
}
func (r *AdminAuditLogRepository) CreateFromOutbox(req models.CreateAdminAuditLogRequest, key string) (uuid.UUID, error) {
	return r.create(req, key)
}
func (r *AdminAuditLogRepository) create(req models.CreateAdminAuditLogRequest, key string) (uuid.UUID, error) {
	query := `
		INSERT INTO admin_audit_log (user_id, user_name, action, details, outbox_deduplication_key)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')) ON CONFLICT (outbox_deduplication_key) WHERE outbox_deduplication_key IS NOT NULL DO NOTHING
		RETURNING id
	`
	var id uuid.UUID
	err := r.db.QueryRow(query, req.UserID, req.UserName, req.Action, req.Details, key).Scan(&id)
	if err == sql.ErrNoRows && key != "" {
		return uuid.Nil, nil
	}
	return id, err
}

// GetAll возвращает список записей журнала с пагинацией и общее количество.
func (r *AdminAuditLogRepository) GetAll(limit, offset int) ([]models.AdminAuditLog, int, error) {
	// Получаем общее количество записей
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM admin_audit_log").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, user_name, action, COALESCE(details, ''), created_at
		FROM admin_audit_log
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]models.AdminAuditLog, 0)
	for rows.Next() {
		var entry models.AdminAuditLog
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.UserName, &entry.Action, &entry.Details, &entry.CreatedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

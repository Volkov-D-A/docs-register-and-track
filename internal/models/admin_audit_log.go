package models

import (
	"time"

	"github.com/google/uuid"
)

// AdminAuditLog представляет собой запись в журнале действий администраторов.
type AdminAuditLog struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	UserName  string    `json:"userName"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateAdminAuditLogRequest описывает внутренний запрос на создание записи в журнале администраторов.
type CreateAdminAuditLogRequest struct {
	UserID   uuid.UUID
	UserName string
	Action   string
	Details  string
}

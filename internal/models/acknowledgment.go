package models

import (
	"time"

	"github.com/google/uuid"
)

// Acknowledgment - задача на ознакомление
type Acknowledgment struct {
	ID             uuid.UUID `json:"-"`
	DocumentID     uuid.UUID `json:"-"`
	DocumentType   string    `json:"documentType"` // 'incoming' или 'outgoing'
	DocumentNumber string    `json:"documentNumber,omitempty"`

	CreatorID   uuid.UUID `json:"-"`
	CreatorName string    `json:"creatorName,omitempty"`

	Content     string     `json:"content"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Пользователи ознакомления
	Users   []AcknowledgmentUser `json:"users,omitempty"`
	UserIDs []string             `json:"userIds,omitempty"` // Для создания
}

type AcknowledgmentUser struct {
	ID               uuid.UUID  `json:"-"`
	AcknowledgmentID uuid.UUID  `json:"-"`
	UserID           uuid.UUID  `json:"-"`
	UserName         string     `json:"userName,omitempty"`
	ViewedAt         *time.Time `json:"viewedAt,omitempty"`
	ConfirmedAt      *time.Time `json:"confirmedAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type AcknowledgmentFilter struct {
	DocumentID   string `json:"documentId,omitempty"`
	UserID       string `json:"userId,omitempty"`
	Status       string `json:"status,omitempty"` // pending, completed
	ShowFinished bool   `json:"showFinished"`
}

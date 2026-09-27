package models

import (
	"time"

	"github.com/google/uuid"
)

// Acknowledgment - задача на ознакомление
type Acknowledgment struct {
	ID             uuid.UUID `json:"-"`
	DocumentID     uuid.UUID `json:"-"`
	DocumentKind   string    `json:"documentKind"` // incoming_letter или outgoing_letter
	DocumentNumber string    `json:"documentNumber,omitempty"`

	CreatorID   uuid.UUID `json:"-"`
	CreatorName string    `json:"creatorName,omitempty"`

	Content     string     `json:"content"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Пользователи ознакомления
	Users []AcknowledgmentUser `json:"users,omitempty"`
}

// AcknowledgmentUser описывает связь пользователя с задачей на ознакомление.
type AcknowledgmentUser struct {
	ID               uuid.UUID  `json:"-"`
	AcknowledgmentID uuid.UUID  `json:"-"`
	UserID           uuid.UUID  `json:"-"`
	UserName         string     `json:"userName,omitempty"`
	ConfirmedAt      *time.Time `json:"confirmedAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// AcknowledgmentFilter описывает параметры фильтрации задач на ознакомление.
type AcknowledgmentFilter struct {
	// AllowedDocumentKinds — серверный scope, не принимается с клиента.
	AllowedDocumentKinds []string `json:"-"`
}

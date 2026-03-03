package models

import (
	"time"

	"github.com/google/uuid"
)

// JournalEntry описывает модель записи в журнале (истории) документа.
type JournalEntry struct {
	ID           uuid.UUID `json:"id"`
	DocumentID   uuid.UUID `json:"documentId"`
	DocumentType string    `json:"documentType"`
	UserID       uuid.UUID `json:"-"`
	UserName     string    `json:"userName,omitempty"`
	Action       string    `json:"action"`
	Details      string    `json:"details"`
	CreatedAt    time.Time `json:"createdAt"`
}

// CreateJournalEntryRequest описывает внутренний запрос на создание записи в журнале.
type CreateJournalEntryRequest struct {
	DocumentID   uuid.UUID
	DocumentType string
	UserID       uuid.UUID
	Action       string
	Details      string
}

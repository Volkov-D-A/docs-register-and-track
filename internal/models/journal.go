package models

import (
	"time"

	"github.com/google/uuid"
)

// JournalEntry описывает модель записи в журнале (истории) документа.
type JournalEntry struct {
	ID        uuid.UUID `json:"id"`
	UserName  string    `json:"userName,omitempty"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateJournalEntryRequest описывает внутренний запрос на создание записи в журнале.
type CreateJournalEntryRequest struct {
	// PreviousReaderIDs preserves final invalidation recipients when implicit access is removed.
	// Stored only in the outbox payload, never in document_journal.
	PreviousReaderIDs []uuid.UUID `json:",omitempty"`
	DocumentID        uuid.UUID
	UserID            uuid.UUID
	Action            string
	Details           string
}

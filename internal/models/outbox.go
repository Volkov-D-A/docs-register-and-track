package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	OutboxEventUserEvent  = "user_event"
	OutboxEventJournal    = "journal_entry"
	OutboxEventAudit      = "admin_audit"
	OutboxEventFileDelete = "attachment_delete"
)

// OutboxEvent is a durable request to perform a side effect after commit.
// Payload is versioned JSON owned by the corresponding event consumer.
type OutboxEvent struct {
	ID                  uuid.UUID
	EventType           string
	DeduplicationKey    string
	Payload             string
	AvailableAt         time.Time
	ProcessingStartedAt *time.Time
	ProcessedAt         *time.Time
	FailedAt            *time.Time
	Attempts            int
	LastError           *string
	CreatedAt           time.Time
}

// OutboxStats is a compact operational view of delivery state.
type OutboxStats struct {
	Pending    int
	Processing int
	Failed     int
	Processed  int
}

// RequiredAuditStats is the delivery state of journal and administrative-audit
// effects. These effects require operator attention when they cannot be sent.
type RequiredAuditStats struct {
	Pending    int
	Processing int
	Failed     int
}

type FailedOutboxEvent struct {
	ID               uuid.UUID `json:"id"`
	EventType        string    `json:"eventType"`
	DeduplicationKey string    `json:"deduplicationKey"`
	Attempts         int       `json:"attempts"`
	LastError        string    `json:"lastError"`
	CreatedAt        time.Time `json:"createdAt"`
	FailedAt         time.Time `json:"failedAt"`
}

type AcknowledgmentConfirmationEffects struct {
	UserEvents []CreateUserEventRequest
}

type AttachmentDeletePayload struct {
	AttachmentID uuid.UUID `json:"attachmentId"`
	StoragePath  string    `json:"storagePath"`
}

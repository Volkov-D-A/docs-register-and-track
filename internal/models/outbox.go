package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	OutboxEventUserEvent = "user_event"
	OutboxEventJournal   = "journal_entry"
	OutboxEventAudit     = "admin_audit"
)

// OutboxEvent is a durable request to perform a side effect after commit.
// Payload is versioned JSON owned by the corresponding event consumer.
// Only delivery inputs are materialized; scheduling and history stay in event_outbox.
type OutboxEvent struct {
	ID               uuid.UUID
	EventType        string
	DeduplicationKey string
	Payload          string
	Attempts         int
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
	ID        uuid.UUID `json:"id"`
	EventType string    `json:"eventType"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"lastError"`
	FailedAt  time.Time `json:"failedAt"`
}

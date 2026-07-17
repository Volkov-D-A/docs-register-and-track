package services

import (
	"encoding/json"
	"fmt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// NewJournalOutboxEvent builds a durable effect. Repositories enqueue it in
// the transaction that changes the corresponding business state.
func NewJournalOutboxEvent(key string, request models.CreateJournalEntryRequest) (models.OutboxEvent, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return models.OutboxEvent{}, err
	}
	return models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: key, Payload: string(payload)}, nil
}

func NewAdminAuditOutboxEvent(key string, request models.CreateAdminAuditLogRequest) (models.OutboxEvent, error) {
	if key == "" {
		return models.OutboxEvent{}, fmt.Errorf("admin audit outbox key is required")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return models.OutboxEvent{}, err
	}
	return models.OutboxEvent{EventType: models.OutboxEventAudit, DeduplicationKey: key, Payload: string(payload)}, nil
}
func NewUserEventOutboxEvent(key string, request models.CreateUserEventRequest) (models.OutboxEvent, error) {
	payload, err := json.Marshal(struct {
		Request models.CreateUserEventRequest `json:"request"`
	}{request})
	if err != nil {
		return models.OutboxEvent{}, err
	}
	return models.OutboxEvent{EventType: models.OutboxEventUserEvent, DeduplicationKey: key, Payload: string(payload)}, nil
}

package services

import (
	"encoding/json"
	"fmt"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
)

// OutboxPublisher is the transitional publisher for legacy services whose
// business repository does not yet expose a shared SQL transaction.
type OutboxPublisher struct{ repo *repository.OutboxRepository }

func NewOutboxPublisher(repo *repository.OutboxRepository) *OutboxPublisher {
	return &OutboxPublisher{repo: repo}
}

func (p *OutboxPublisher) PublishJournal(key string, request models.CreateJournalEntryRequest) error {
	if p == nil {
		return nil
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	return p.repo.Enqueue(models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: key, Payload: string(payload)})
}

func (p *OutboxPublisher) PublishAdminAudit(key string, request models.CreateAdminAuditLogRequest) error {
	if p == nil {
		return nil
	}
	if key == "" {
		return fmt.Errorf("admin audit outbox key is required")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	return p.repo.Enqueue(models.OutboxEvent{EventType: models.OutboxEventAudit, DeduplicationKey: key, Payload: string(payload)})
}
func (p *OutboxPublisher) PublishUserEvent(key string, request models.CreateUserEventRequest) error {
	if p == nil {
		return nil
	}
	payload, err := json.Marshal(struct {
		Request models.CreateUserEventRequest `json:"request"`
	}{request})
	if err != nil {
		return err
	}
	return p.repo.Enqueue(models.OutboxEvent{EventType: models.OutboxEventUserEvent, DeduplicationKey: key, Payload: string(payload)})
}

package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/google/uuid"
)

type userEventPayload struct {
	Request models.CreateUserEventRequest `json:"request"`
}

type FileDeleter interface {
	DeleteFile(ctx context.Context, objectName string) error
}

type Worker struct {
	outbox            *repository.OutboxRepository
	events            *repository.UserEventRepository
	journal           *repository.JournalRepository
	audit             *repository.AdminAuditLogRepository
	attachments       *repository.AttachmentRepository
	storage           FileDeleter
	lastRequiredAudit models.RequiredAuditStats
}

const (
	maxAttempts       = 10
	maxRetryDelay     = time.Hour
	queueAlertSize    = 100
	staleClaimTimeout = 5 * time.Minute
	consumerTimeout   = 30 * time.Second
)

func NewWorker(outbox *repository.OutboxRepository, events *repository.UserEventRepository, journal *repository.JournalRepository, audit *repository.AdminAuditLogRepository, attachments *repository.AttachmentRepository, storage FileDeleter) *Worker {
	return &Worker{outbox: outbox, events: events, journal: journal, audit: audit, attachments: attachments, storage: storage}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		// A crashed process can leave a claimed task behind. Reaping on every
		// polling iteration, rather than just at startup, also recovers claims
		// from a stalled concurrent worker.
		if err := w.outbox.ReleaseStaleClaims(time.Now().Add(-staleClaimTimeout)); err != nil {
			slog.Warn("failed to release stale outbox claims", "error", err)
		}
		if err := w.ProcessOnceContext(ctx); err != nil {
			slog.Warn("outbox processing failed", "error", err)
		}
		w.observeRequiredAudit()
		w.observeQueue()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) observeQueue() {
	stats, err := w.outbox.Stats()
	if err != nil {
		slog.Warn("failed to read outbox queue state", "error", err)
		return
	}
	if stats.Failed > 0 {
		slog.Error("outbox has terminal failures", "failed", stats.Failed, "pending", stats.Pending, "processing", stats.Processing)
	}
	if stats.Pending >= queueAlertSize {
		slog.Warn("outbox queue exceeds alert threshold", "pending", stats.Pending, "threshold", queueAlertSize)
	}
}

// observeRequiredAudit emits a state-change alert for effects that represent
// business and administrative audit. Keeping the previous state avoids a log
// message every polling interval while still making a restart visible.
func (w *Worker) observeRequiredAudit() {
	stats, err := w.outbox.RequiredAuditStats()
	if err != nil {
		slog.Warn("failed to read required outbox audit state", "error", err)
		return
	}
	if stats == w.lastRequiredAudit {
		return
	}
	if stats.Failed > 0 {
		slog.Error("required outbox audit events have terminal failures", "pending", stats.Pending, "processing", stats.Processing, "failed", stats.Failed)
	} else if stats.Pending+stats.Processing > 0 {
		slog.Warn("required outbox audit events await delivery", "pending", stats.Pending, "processing", stats.Processing)
	}
	w.lastRequiredAudit = stats
}

func (w *Worker) ProcessOnce() error {
	return w.ProcessOnceContext(context.Background())
}

// ProcessOnceContext delivers one claimed batch while propagating shutdown
// and per-consumer deadlines to operations that support a context.
func (w *Worker) ProcessOnceContext(ctx context.Context) error {
	events, err := w.outbox.ClaimPending(50)
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := w.process(ctx, event); err != nil {
			if markErr := w.outbox.MarkFailed(event.ID, event.Attempts, retryDelay(event.Attempts), maxAttempts, err.Error()); markErr != nil {
				return markErr
			}
			continue
		}
		if err := w.outbox.MarkProcessed(event.ID); err != nil {
			return err
		}
	}
	return nil
}

func retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		return time.Second
	}
	delay := time.Second * time.Duration(1<<min(attempts-1, 16))
	if delay > maxRetryDelay {
		return maxRetryDelay
	}
	return delay
}

func (w *Worker) process(parent context.Context, event models.OutboxEvent) error {
	ctx, cancel := context.WithTimeout(parent, consumerTimeout)
	defer cancel()

	switch event.EventType {
	case models.OutboxEventUserEvent:
		var payload userEventPayload
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return fmt.Errorf("invalid user_event payload: %w", err)
		}
		return w.events.CreateFromOutbox(payload.Request, event.DeduplicationKey)
	case models.OutboxEventJournal:
		var payload models.CreateJournalEntryRequest
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return fmt.Errorf("invalid journal payload: %w", err)
		}
		_, err := w.journal.CreateFromOutbox(ctx, payload, event.DeduplicationKey)
		return err
	case models.OutboxEventAudit:
		var payload models.CreateAdminAuditLogRequest
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return fmt.Errorf("invalid admin_audit payload: %w", err)
		}
		_, err := w.audit.CreateFromOutbox(payload, event.DeduplicationKey)
		return err
	case models.OutboxEventFileDelete:
		if w.storage == nil || w.attachments == nil {
			return fmt.Errorf("attachment deletion consumer is not configured")
		}
		var payload models.AttachmentDeletePayload
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return fmt.Errorf("invalid attachment_delete payload: %w", err)
		}
		if payload.AttachmentID == uuid.Nil || payload.StoragePath == "" {
			return fmt.Errorf("invalid attachment_delete payload")
		}
		if err := w.storage.DeleteFile(ctx, payload.StoragePath); err != nil {
			return err
		}
		return w.attachments.DeleteMarked(payload.AttachmentID)
	default:
		return fmt.Errorf("unsupported outbox event type %q", event.EventType)
	}
}

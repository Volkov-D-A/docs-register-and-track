package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
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
	metrics           *observability.Registry
	now               func() time.Time
	options           Options
}

const (
	maxAttempts       = 10
	maxRetryDelay     = time.Hour
	queueAlertSize    = 100
	cleanupBatchSize  = 1000
	cleanupMaxBatches = 10
)

type Options struct {
	PollingInterval    time.Duration
	BatchSize          int
	StaleClaimTimeout  time.Duration
	ConsumerTimeout    time.Duration
	ProcessedRetention time.Duration
	CleanupInterval    time.Duration
}

func DefaultOptions() Options {
	return Options{
		PollingInterval:    5 * time.Second,
		BatchSize:          50,
		StaleClaimTimeout:  5 * time.Minute,
		ConsumerTimeout:    30 * time.Second,
		ProcessedRetention: 90 * 24 * time.Hour,
		CleanupInterval:    time.Hour,
	}
}

func (o Options) WithDefaults() Options {
	defaults := DefaultOptions()
	if o.PollingInterval == 0 {
		o.PollingInterval = defaults.PollingInterval
	}
	if o.BatchSize == 0 {
		o.BatchSize = defaults.BatchSize
	}
	if o.StaleClaimTimeout == 0 {
		o.StaleClaimTimeout = defaults.StaleClaimTimeout
	}
	if o.ConsumerTimeout == 0 {
		o.ConsumerTimeout = defaults.ConsumerTimeout
	}
	if o.ProcessedRetention == 0 {
		o.ProcessedRetention = defaults.ProcessedRetention
	}
	if o.CleanupInterval == 0 {
		o.CleanupInterval = defaults.CleanupInterval
	}
	return o
}

func (o Options) Validate() error {
	if o.PollingInterval < time.Second || o.PollingInterval > time.Minute {
		return fmt.Errorf("outbox polling interval must be between 1s and 1m")
	}
	if o.BatchSize < 1 || o.BatchSize > 1000 {
		return fmt.Errorf("outbox batch size must be between 1 and 1000")
	}
	if o.StaleClaimTimeout < time.Minute || o.StaleClaimTimeout > time.Hour {
		return fmt.Errorf("outbox stale claim timeout must be between 1m and 1h")
	}
	if o.ConsumerTimeout < time.Second || o.ConsumerTimeout > 10*time.Minute {
		return fmt.Errorf("outbox consumer timeout must be between 1s and 10m")
	}
	if o.ProcessedRetention < 24*time.Hour || o.ProcessedRetention > 365*24*time.Hour {
		return fmt.Errorf("outbox processed retention must be between 1 and 365 days")
	}
	if o.CleanupInterval < time.Minute || o.CleanupInterval > 24*time.Hour {
		return fmt.Errorf("outbox cleanup interval must be between 1m and 24h")
	}
	return nil
}

func NewWorker(outbox *repository.OutboxRepository, events *repository.UserEventRepository, journal *repository.JournalRepository, audit *repository.AdminAuditLogRepository, attachments *repository.AttachmentRepository, storage FileDeleter) *Worker {
	worker, err := NewWorkerWithOptions(outbox, events, journal, audit, attachments, storage, DefaultOptions())
	if err != nil {
		panic(err)
	}
	return worker
}

func NewWorkerWithOptions(outbox *repository.OutboxRepository, events *repository.UserEventRepository, journal *repository.JournalRepository, audit *repository.AdminAuditLogRepository, attachments *repository.AttachmentRepository, storage FileDeleter, options Options) (*Worker, error) {
	options = options.WithDefaults()
	if err := options.Validate(); err != nil {
		return nil, err
	}
	if outbox == nil {
		return nil, fmt.Errorf("outbox repository is required")
	}
	return &Worker{outbox: outbox, events: events, journal: journal, audit: audit, attachments: attachments, storage: storage, now: time.Now, options: options}, nil
}

func (w *Worker) SetMetrics(metrics *observability.Registry) { w.metrics = metrics }

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.options.PollingInterval)
	defer ticker.Stop()
	cleanupTicker := time.NewTicker(w.options.CleanupInterval)
	defer cleanupTicker.Stop()
	cleanupDue := true
	for {
		if cleanupDue && ctx.Err() == nil {
			w.cleanupProcessed(ctx)
			cleanupDue = false
		}
		// A crashed process can leave a claimed task behind. Reaping on every
		// polling iteration, rather than just at startup, also recovers claims
		// from a stalled concurrent worker.
		if err := w.outbox.ReleaseStaleClaims(time.Now().Add(-w.options.StaleClaimTimeout)); err != nil {
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
		case <-cleanupTicker.C:
			cleanupDue = true
		}
	}
}

func (w *Worker) cleanupProcessed(ctx context.Context) {
	cutoff := w.now().Add(-w.options.ProcessedRetention)
	var total int64
	for range cleanupMaxBatches {
		deleted, err := w.outbox.DeleteProcessedBefore(ctx, cutoff, cleanupBatchSize)
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("failed to clean processed outbox events", "error", err)
			}
			return
		}
		total += deleted
		if deleted < cleanupBatchSize {
			break
		}
	}
	if total > 0 {
		slog.Info("cleaned processed outbox events", "deleted", total, "retention_days", int(w.options.ProcessedRetention/(24*time.Hour)))
	}
}

func (w *Worker) observeQueue() {
	stats, err := w.outbox.QueueStats()
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
	if w.metrics != nil {
		w.metrics.SetGauge("outbox.pending", float64(stats.Pending))
		w.metrics.SetGauge("outbox.processing", float64(stats.Processing))
		w.metrics.SetGauge("outbox.failed", float64(stats.Failed))
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
	events, err := w.outbox.ClaimPending(w.options.BatchSize)
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := w.process(ctx, event); err != nil {
			if w.metrics != nil {
				w.metrics.AddCounter("outbox.retries", 1)
			}
			if markErr := w.outbox.MarkFailed(event.ID, event.Attempts, retryDelay(event.Attempts), maxAttempts, err.Error()); markErr != nil {
				return markErr
			}
			continue
		}
		if err := w.outbox.MarkProcessed(event.ID); err != nil {
			return err
		}
		if w.metrics != nil {
			w.metrics.AddCounter("outbox.processed", 1)
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
	ctx, cancel := context.WithTimeout(parent, w.options.ConsumerTimeout)
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
		mutation, err := w.attachments.BeginStorageMutation(ctx)
		if err != nil {
			return fmt.Errorf("failed to coordinate attachment deletion: %w", err)
		}
		if err := w.storage.DeleteFile(mutation.Context(), payload.StoragePath); err != nil {
			_ = mutation.Finish()
			return err
		}
		if err := w.attachments.DeleteMarkedAndDecrementStorageStatistics(payload.AttachmentID); err != nil {
			_ = mutation.Finish()
			return err
		}
		return mutation.Finish()
	default:
		return fmt.Errorf("unsupported outbox event type %q", event.EventType)
	}
}

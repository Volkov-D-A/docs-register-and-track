package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// OutboxRepository persists side-effect requests in the same transaction as
// their business change. Delivery is implemented separately by a worker.
type OutboxRepository struct{ db *database.DB }

func NewOutboxRepository(db *database.DB) *OutboxRepository { return &OutboxRepository{db: db} }

func (r *OutboxRepository) EnqueueTx(tx *sql.Tx, event models.OutboxEvent) error {
	if event.EventType == "" || event.DeduplicationKey == "" || event.Payload == "" {
		return fmt.Errorf("outbox event type, deduplication key and payload are required")
	}
	_, err := tx.Exec(`INSERT INTO event_outbox (event_type, deduplication_key, payload)
		VALUES ($1, $2, $3::jsonb) ON CONFLICT (deduplication_key) DO NOTHING`,
		event.EventType, event.DeduplicationKey, event.Payload)
	return err
}

func (r *OutboxRepository) Enqueue(event models.OutboxEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := r.EnqueueTx(tx, event); err != nil {
		return err
	}
	return tx.Commit()
}

// ClaimPending atomically reserves due events for one worker. SKIP LOCKED
// allows several application instances to process independent events safely.
func (r *OutboxRepository) ClaimPending(limit int) ([]models.OutboxEvent, error) {
	if limit < 1 {
		return []models.OutboxEvent{}, nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`WITH due AS (
		SELECT id FROM event_outbox
		WHERE processed_at IS NULL AND failed_at IS NULL AND processing_started_at IS NULL AND available_at <= CURRENT_TIMESTAMP
		ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1
	)
	UPDATE event_outbox e SET processing_started_at = CURRENT_TIMESTAMP, attempts = attempts + 1
	FROM due WHERE e.id = due.id
	RETURNING e.id, e.event_type, e.deduplication_key, e.payload::text, e.available_at,
		e.processing_started_at, e.processed_at, e.failed_at, e.attempts, e.last_error, e.created_at`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]models.OutboxEvent, 0)
	for rows.Next() {
		var event models.OutboxEvent
		if err := rows.Scan(&event.ID, &event.EventType, &event.DeduplicationKey, &event.Payload, &event.AvailableAt,
			&event.ProcessingStartedAt, &event.ProcessedAt, &event.FailedAt, &event.Attempts, &event.LastError, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *OutboxRepository) MarkProcessed(id uuid.UUID) error {
	_, err := r.db.Exec(`UPDATE event_outbox SET processed_at = CURRENT_TIMESTAMP, processing_started_at = NULL, last_error = NULL
		WHERE id = $1 AND processed_at IS NULL`, id)
	return err
}

// MarkFailed applies bounded exponential retry. At maxAttempts the task becomes
// terminal and is excluded from automatic claiming until Requeue is called.
func (r *OutboxRepository) MarkFailed(id uuid.UUID, attempts int, delay time.Duration, maxAttempts int, message string) error {
	if attempts < 1 {
		attempts = 1
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	_, err := r.db.Exec(`UPDATE event_outbox
		SET processing_started_at = NULL, last_error = $5,
			failed_at = CASE WHEN $2 >= $3 THEN CURRENT_TIMESTAMP ELSE NULL END,
			available_at = CASE WHEN $2 >= $3 THEN available_at ELSE CURRENT_TIMESTAMP + ($4 * INTERVAL '1 second') END
		WHERE id = $1 AND processed_at IS NULL`, id, attempts, maxAttempts, delay.Seconds(), message)
	return err
}

// Requeue is the explicit administrative action for a terminal failure.
func (r *OutboxRepository) Requeue(id uuid.UUID) error {
	_, err := r.db.Exec(`UPDATE event_outbox
		SET processing_started_at = NULL, failed_at = NULL, last_error = NULL, attempts = 0, available_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND processed_at IS NULL AND failed_at IS NOT NULL`, id)
	return err
}

func (r *OutboxRepository) Stats() (models.OutboxStats, error) {
	var stats models.OutboxStats
	err := r.db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE processed_at IS NULL AND failed_at IS NULL AND processing_started_at IS NULL),
		COUNT(*) FILTER (WHERE processed_at IS NULL AND failed_at IS NULL AND processing_started_at IS NOT NULL),
		COUNT(*) FILTER (WHERE failed_at IS NOT NULL),
		COUNT(*) FILTER (WHERE processed_at IS NOT NULL)
		FROM event_outbox`).Scan(&stats.Pending, &stats.Processing, &stats.Failed, &stats.Processed)
	return stats, err
}

func (r *OutboxRepository) RequiredAuditStats() (models.RequiredAuditStats, error) {
	var stats models.RequiredAuditStats
	err := r.db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE processed_at IS NULL AND failed_at IS NULL AND processing_started_at IS NULL),
		COUNT(*) FILTER (WHERE processed_at IS NULL AND failed_at IS NULL AND processing_started_at IS NOT NULL),
		COUNT(*) FILTER (WHERE failed_at IS NOT NULL)
		FROM event_outbox
		WHERE event_type IN ($1, $2)`, models.OutboxEventJournal, models.OutboxEventAudit).
		Scan(&stats.Pending, &stats.Processing, &stats.Failed)
	return stats, err
}

func (r *OutboxRepository) GetFailed(limit int) ([]models.FailedOutboxEvent, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(`SELECT id, event_type, deduplication_key, attempts, COALESCE(last_error, ''), created_at, failed_at
		FROM event_outbox WHERE failed_at IS NOT NULL ORDER BY failed_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]models.FailedOutboxEvent, 0)
	for rows.Next() {
		var event models.FailedOutboxEvent
		if err := rows.Scan(&event.ID, &event.EventType, &event.DeduplicationKey, &event.Attempts, &event.LastError, &event.CreatedAt, &event.FailedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// ReleaseStaleClaims makes events retryable after a process crash.
func (r *OutboxRepository) ReleaseStaleClaims(before time.Time) error {
	_, err := r.db.Exec(`UPDATE event_outbox SET processing_started_at = NULL
		WHERE processed_at IS NULL AND failed_at IS NULL AND processing_started_at < $1`, before)
	return err
}

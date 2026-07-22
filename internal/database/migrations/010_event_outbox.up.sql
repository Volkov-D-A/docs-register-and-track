CREATE TABLE event_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    deduplication_key VARCHAR(255) NOT NULL UNIQUE,
    payload JSONB NOT NULL,
    available_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processing_started_at TIMESTAMP WITH TIME ZONE,
    processed_at TIMESTAMP WITH TIME ZONE,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error TEXT,
    failed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_event_outbox_pending
    ON event_outbox (available_at, created_at)
    WHERE processed_at IS NULL;

CREATE INDEX idx_event_outbox_failed
    ON event_outbox (failed_at, created_at)
    WHERE failed_at IS NOT NULL;

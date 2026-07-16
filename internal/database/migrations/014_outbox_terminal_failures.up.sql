ALTER TABLE event_outbox
    ADD COLUMN failed_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX idx_event_outbox_failed
    ON event_outbox (failed_at, created_at)
    WHERE failed_at IS NOT NULL;

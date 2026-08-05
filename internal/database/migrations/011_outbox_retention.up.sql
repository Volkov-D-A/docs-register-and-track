CREATE INDEX idx_event_outbox_processed_cleanup
    ON event_outbox (processed_at, id)
    WHERE processed_at IS NOT NULL;

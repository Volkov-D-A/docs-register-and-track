ALTER TABLE user_events ADD COLUMN outbox_deduplication_key VARCHAR(255);
CREATE UNIQUE INDEX idx_user_events_outbox_deduplication
    ON user_events (outbox_deduplication_key)
    WHERE outbox_deduplication_key IS NOT NULL;

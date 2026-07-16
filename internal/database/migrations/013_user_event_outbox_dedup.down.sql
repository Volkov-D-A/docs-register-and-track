DROP INDEX IF EXISTS idx_user_events_outbox_deduplication;
ALTER TABLE user_events DROP COLUMN IF EXISTS outbox_deduplication_key;

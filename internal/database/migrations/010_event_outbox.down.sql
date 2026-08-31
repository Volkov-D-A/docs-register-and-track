DROP INDEX IF EXISTS idx_event_outbox_pending;
DROP INDEX IF EXISTS idx_event_outbox_failed;
DROP INDEX IF EXISTS idx_event_outbox_processed_cleanup;
DROP TABLE IF EXISTS event_outbox;

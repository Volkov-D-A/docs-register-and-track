DROP INDEX IF EXISTS idx_attachments_pending_deletion;

ALTER TABLE attachments
    DROP COLUMN IF EXISTS deletion_requested_at;

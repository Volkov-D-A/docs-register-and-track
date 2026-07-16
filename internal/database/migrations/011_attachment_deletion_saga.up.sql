-- Keep a durable deletion intent while the object is removed from MinIO.
-- PostgreSQL and MinIO cannot participate in one transaction, so this state
-- prevents an attachment whose object was already deleted from being visible.
ALTER TABLE attachments
    ADD COLUMN deletion_requested_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX idx_attachments_pending_deletion
    ON attachments (deletion_requested_at)
    WHERE deletion_requested_at IS NOT NULL;

DROP INDEX IF EXISTS idx_admin_audit_log_outbox_dedup;
ALTER TABLE admin_audit_log DROP COLUMN IF EXISTS outbox_deduplication_key;
DROP INDEX IF EXISTS idx_document_journal_outbox_dedup;
ALTER TABLE document_journal DROP COLUMN IF EXISTS outbox_deduplication_key;

ALTER TABLE document_journal ADD COLUMN outbox_deduplication_key VARCHAR(255);
CREATE UNIQUE INDEX idx_document_journal_outbox_dedup ON document_journal (outbox_deduplication_key) WHERE outbox_deduplication_key IS NOT NULL;
ALTER TABLE admin_audit_log ADD COLUMN outbox_deduplication_key VARCHAR(255);
CREATE UNIQUE INDEX idx_admin_audit_log_outbox_dedup ON admin_audit_log (outbox_deduplication_key) WHERE outbox_deduplication_key IS NOT NULL;

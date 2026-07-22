DROP INDEX IF EXISTS idx_document_journal_created_at;
DROP INDEX IF EXISTS idx_document_journal_doc_id;
DROP INDEX IF EXISTS idx_document_journal_outbox_dedup;

DROP TABLE IF EXISTS document_journal;

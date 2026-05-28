DROP INDEX IF EXISTS idx_documents_created_by_kind_idempotency;

ALTER TABLE documents
    DROP COLUMN IF EXISTS idempotency_key;

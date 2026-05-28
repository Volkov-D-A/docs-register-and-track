ALTER TABLE documents
    ADD COLUMN idempotency_key UUID;

UPDATE documents
SET idempotency_key = gen_random_uuid()
WHERE idempotency_key IS NULL;

ALTER TABLE documents
    ALTER COLUMN idempotency_key SET NOT NULL;

CREATE UNIQUE INDEX idx_documents_created_by_kind_idempotency
    ON documents (created_by, kind, idempotency_key);

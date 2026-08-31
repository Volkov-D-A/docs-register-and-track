CREATE TABLE document_command_idempotency (
    principal_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation TEXT NOT NULL,
    idempotency_key UUID NOT NULL,
    request_hash TEXT NOT NULL,
    document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (principal_id, operation, idempotency_key)
);

CREATE INDEX idx_document_command_idempotency_created_at
    ON document_command_idempotency(created_at);

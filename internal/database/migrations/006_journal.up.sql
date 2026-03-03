CREATE TABLE IF NOT EXISTS document_journal (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL,
    document_type VARCHAR(50) NOT NULL, -- 'incoming' or 'outgoing'
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL, -- 'CREATE', 'UPDATE', 'DELETE', 'STATUS_CHANGE', etc.
    details TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_document_journal_doc_id ON document_journal(document_id, document_type);
CREATE INDEX IF NOT EXISTS idx_document_journal_created_at ON document_journal(created_at DESC);

CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    document_id UUID NOT NULL,
    document_type VARCHAR(50) NOT NULL, -- 'incoming' or 'outgoing'
    executor_id UUID NOT NULL REFERENCES users (id),
    content TEXT NOT NULL,
    deadline DATE,
    status VARCHAR(50) NOT NULL DEFAULT 'new', -- new, in_progress, completed, cancelled
    report TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_assignments_executor ON assignments (executor_id);

CREATE INDEX idx_assignments_document ON assignments (document_id);
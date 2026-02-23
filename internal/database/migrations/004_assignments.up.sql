-- 11. Assignments
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
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_assignments_executor ON assignments (executor_id);

CREATE INDEX idx_assignments_document ON assignments (document_id);

-- 14. Assignment Co-Executors
CREATE TABLE assignment_co_executors (
    assignment_id UUID NOT NULL REFERENCES assignments (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (assignment_id, user_id)
);

CREATE INDEX idx_assignment_co_executors_assignment ON assignment_co_executors (assignment_id);

CREATE INDEX idx_assignment_co_executors_user ON assignment_co_executors (user_id);
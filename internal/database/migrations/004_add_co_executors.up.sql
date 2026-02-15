CREATE TABLE assignment_co_executors (
    assignment_id UUID NOT NULL REFERENCES assignments (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (assignment_id, user_id)
);

CREATE INDEX idx_assignment_co_executors_assignment ON assignment_co_executors (assignment_id);

CREATE INDEX idx_assignment_co_executors_user ON assignment_co_executors (user_id);
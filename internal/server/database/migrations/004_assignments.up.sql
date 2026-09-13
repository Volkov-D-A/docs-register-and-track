-- 11. Recurring assignment series
CREATE TABLE assignment_series (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    executor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    content TEXT NOT NULL,
    interval_unit VARCHAR(20) NOT NULL CHECK (interval_unit IN ('day', 'week', 'month', 'year')),
    interval_value INTEGER NOT NULL CHECK (interval_value BETWEEN 1 AND 3650),
    day_rule VARCHAR(20) NOT NULL CHECK (day_rule IN ('same_day', 'fixed', 'last_day')),
    day_of_month INTEGER CHECK (
        (interval_unit IN ('day', 'week') AND day_rule = 'same_day' AND day_of_month IS NULL)
        OR (interval_unit IN ('month', 'year') AND day_rule = 'fixed' AND day_of_month BETWEEN 1 AND 31)
        OR (interval_unit IN ('month', 'year') AND day_rule = 'last_day' AND day_of_month IS NULL)
    ),
    current_assignment_id UUID,
    current_iteration INTEGER NOT NULL DEFAULT 1 CHECK (current_iteration > 0),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    cancelled_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE assignment_series_co_executors (
    series_id UUID NOT NULL REFERENCES assignment_series(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    PRIMARY KEY (series_id, user_id)
);

CREATE INDEX idx_assignment_series_document ON assignment_series(document_id);
CREATE INDEX idx_assignment_series_current ON assignment_series(current_assignment_id);
CREATE INDEX idx_assignment_series_active ON assignment_series(active) WHERE active = TRUE;

-- 12. Assignments
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    document_id UUID NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    executor_id UUID NOT NULL REFERENCES users (id),
    content TEXT NOT NULL,
    deadline DATE,
    status VARCHAR(50) NOT NULL DEFAULT 'new', -- new, in_progress, completed, cancelled
    report TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    series_id UUID REFERENCES assignment_series(id) ON DELETE CASCADE,
    iteration_number INTEGER,
    planned_deadline DATE,
    CONSTRAINT assignments_series_iteration_consistency CHECK (
        (series_id IS NULL AND iteration_number IS NULL)
        OR (series_id IS NOT NULL AND iteration_number IS NOT NULL AND iteration_number > 0)
    ),
    CONSTRAINT assignments_series_iteration_unique UNIQUE (series_id, iteration_number)
);

CREATE INDEX idx_assignments_executor ON assignments (executor_id);

CREATE INDEX idx_assignments_document ON assignments (document_id);
CREATE INDEX idx_assignments_document_executor ON assignments (document_id, executor_id);

CREATE INDEX idx_assignments_deadline ON assignments (deadline);
CREATE INDEX idx_assignments_executor_status ON assignments (executor_id, status);
CREATE INDEX idx_assignments_active_deadline ON assignments (deadline, status)
    WHERE status IN ('new', 'in_progress') AND deadline IS NOT NULL;
CREATE INDEX idx_assignments_series ON assignments(series_id, iteration_number DESC);

ALTER TABLE assignment_series
    ADD CONSTRAINT assignment_series_current_assignment_fk
    FOREIGN KEY (current_assignment_id) REFERENCES assignments(id) ON DELETE SET NULL;

-- 13. Assignment Co-Executors
CREATE TABLE assignment_co_executors (
    assignment_id UUID NOT NULL REFERENCES assignments (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (assignment_id, user_id)
);

CREATE INDEX idx_assignment_co_executors_assignment ON assignment_co_executors (assignment_id);

CREATE INDEX idx_assignment_co_executors_user ON assignment_co_executors (user_id);
CREATE INDEX idx_assignment_co_executors_user_assignment ON assignment_co_executors (user_id, assignment_id);

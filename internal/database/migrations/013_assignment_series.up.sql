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

ALTER TABLE assignments
    ADD COLUMN series_id UUID REFERENCES assignment_series(id) ON DELETE CASCADE,
    ADD COLUMN iteration_number INTEGER,
    ADD COLUMN planned_deadline DATE;

ALTER TABLE assignments
    ADD CONSTRAINT assignments_series_iteration_consistency CHECK (
        (series_id IS NULL AND iteration_number IS NULL)
        OR (series_id IS NOT NULL AND iteration_number IS NOT NULL AND iteration_number > 0)
    ),
    ADD CONSTRAINT assignments_series_iteration_unique UNIQUE (series_id, iteration_number);

ALTER TABLE assignment_series
    ADD CONSTRAINT assignment_series_current_assignment_fk
    FOREIGN KEY (current_assignment_id) REFERENCES assignments(id) ON DELETE SET NULL;

CREATE INDEX idx_assignment_series_document ON assignment_series(document_id);
CREATE INDEX idx_assignment_series_current ON assignment_series(current_assignment_id);
CREATE INDEX idx_assignment_series_active ON assignment_series(active) WHERE active = TRUE;
CREATE INDEX idx_assignments_series ON assignments(series_id, iteration_number DESC);

ALTER TABLE attachments
    ADD COLUMN assignment_id UUID REFERENCES assignments(id) ON DELETE SET NULL;
CREATE INDEX idx_attachments_assignment ON attachments(assignment_id)
    WHERE assignment_id IS NOT NULL;

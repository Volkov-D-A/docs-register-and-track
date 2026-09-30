CREATE TABLE report_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request JSONB NOT NULL,
    frequency TEXT NOT NULL CHECK (frequency IN ('weekly', 'monthly')),
    timezone TEXT NOT NULL,
    format TEXT NOT NULL CHECK (format IN ('xlsx', 'pdf')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    next_run_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX report_schedules_due ON report_schedules(next_run_at) WHERE enabled;

CREATE TABLE report_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES report_schedules(id) ON DELETE CASCADE,
    planned_at TIMESTAMPTZ NOT NULL,
    request JSONB NOT NULL,
    format TEXT NOT NULL,
    definition_version INT NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'ready', 'failed')),
    lease_until TIMESTAMPTZ,
    storage_key TEXT,
    file_size BIGINT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    UNIQUE(schedule_id, planned_at)
);
CREATE INDEX report_runs_work ON report_runs(status, lease_until, created_at);
CREATE INDEX report_runs_schedule ON report_runs(schedule_id, planned_at DESC);

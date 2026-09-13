CREATE TABLE backup_settings (
    id BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),
    settings JSONB NOT NULL
);
CREATE TABLE backup_jobs (
    id TEXT PRIMARY KEY,
    state JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE backup_audit (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

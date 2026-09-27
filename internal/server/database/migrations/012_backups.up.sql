CREATE TABLE backup_settings (
    id BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),
    settings JSONB NOT NULL
);

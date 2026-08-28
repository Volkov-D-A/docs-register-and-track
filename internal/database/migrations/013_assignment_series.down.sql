ALTER TABLE assignment_series DROP CONSTRAINT IF EXISTS assignment_series_current_assignment_fk;

DROP INDEX IF EXISTS idx_attachments_assignment;
ALTER TABLE attachments DROP COLUMN IF EXISTS assignment_id;

ALTER TABLE assignments
    DROP CONSTRAINT IF EXISTS assignments_series_iteration_unique,
    DROP CONSTRAINT IF EXISTS assignments_series_iteration_consistency,
    DROP COLUMN IF EXISTS planned_deadline,
    DROP COLUMN IF EXISTS iteration_number,
    DROP COLUMN IF EXISTS series_id;

DROP TABLE IF EXISTS assignment_series_co_executors;
DROP TABLE IF EXISTS assignment_series;

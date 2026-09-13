ALTER TABLE assignment_series DROP CONSTRAINT IF EXISTS assignment_series_current_assignment_fk;

DROP TABLE IF EXISTS assignment_co_executors;

DROP TABLE IF EXISTS assignments;

DROP TABLE IF EXISTS assignment_series_co_executors;

DROP TABLE IF EXISTS assignment_series;

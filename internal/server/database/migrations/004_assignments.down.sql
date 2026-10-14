DROP TRIGGER IF EXISTS deadline_co_executor_deletion ON deadline_executors;
DROP FUNCTION IF EXISTS restrict_deadline_co_executor_deletion();
ALTER TABLE IF EXISTS document_deadline_series DROP CONSTRAINT IF EXISTS deadline_series_current_control;
DROP TABLE IF EXISTS document_deadline_controls;
DROP TABLE IF EXISTS document_deadline_series;
DROP TABLE IF EXISTS assignment_recipients;

ALTER TABLE assignment_series DROP CONSTRAINT IF EXISTS assignment_series_current_assignment_fk;

DROP TABLE IF EXISTS assignment_co_executors;

DROP TABLE IF EXISTS assignments;

DROP TABLE IF EXISTS assignment_series_co_executors;

DROP TABLE IF EXISTS assignment_series;

DROP FUNCTION IF EXISTS verify_deadline_co_executors();
DROP FUNCTION IF EXISTS deadline_executor_set_valid(UUID,UUID[]);

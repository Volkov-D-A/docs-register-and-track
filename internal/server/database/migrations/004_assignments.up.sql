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
    executor_id UUID REFERENCES users (id),
    type TEXT NOT NULL DEFAULT 'execution' CHECK (type IN ('execution', 'acknowledgment')),
    creator_id UUID REFERENCES users(id),
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
    CONSTRAINT assignments_type_consistency CHECK (
        (type = 'execution' AND executor_id IS NOT NULL)
        OR (type = 'acknowledgment' AND executor_id IS NULL AND creator_id IS NOT NULL
            AND series_id IS NULL AND report IS NULL AND status IN ('new', 'finished'))
    ),
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

CREATE INDEX idx_assignments_type_document ON assignments(type, document_id);

CREATE TABLE assignment_recipients (
    id UUID PRIMARY KEY,
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    confirmed_at TIMESTAMPTZ,
    confirmed_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assignment_id, user_id)
);
CREATE INDEX idx_assignment_recipients_user_assignment ON assignment_recipients(user_id, assignment_id);
CREATE INDEX idx_assignment_recipients_pending_user ON assignment_recipients(user_id, assignment_id)
    WHERE confirmed_at IS NULL;

-- New deadline model; legacy consumers are removed in the subsequent UI stage.
CREATE TABLE document_deadline_series (
 id UUID PRIMARY KEY,
 document_id UUID NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
 created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 executor_id UUID NOT NULL REFERENCES deadline_executors(id) ON DELETE RESTRICT,
 co_executor_ids UUID[] NOT NULL DEFAULT '{}',
 subject TEXT NOT NULL CHECK (length(btrim(subject)) <= 2000),
 interval_unit TEXT NOT NULL CHECK (interval_unit IN ('day','week','month','year')),
 interval_value INTEGER NOT NULL CHECK (interval_value BETWEEN 1 AND 3650),
 day_rule TEXT NOT NULL,
 day_of_month INTEGER NOT NULL DEFAULT 0,
 first_due_date DATE NOT NULL,
 active BOOLEAN NOT NULL DEFAULT true,
 stop_reason TEXT NOT NULL DEFAULT '',
 current_control_id UUID NOT NULL,
 current_iteration INTEGER NOT NULL CHECK (current_iteration > 0),
 version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
 CHECK ((interval_unit IN ('day','week') AND day_rule='same_day' AND day_of_month=0)
    OR (interval_unit IN ('month','year') AND day_rule='fixed' AND day_of_month BETWEEN 1 AND 31)
    OR (interval_unit IN ('month','year') AND day_rule='last_day' AND day_of_month=0))
);
CREATE TABLE document_deadline_controls (
 id UUID PRIMARY KEY,
 document_id UUID NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
 created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 executor_id UUID NOT NULL REFERENCES deadline_executors(id) ON DELETE RESTRICT,
 co_executor_ids UUID[] NOT NULL DEFAULT '{}',
 subject TEXT NOT NULL CHECK (length(btrim(subject)) <= 2000),
 due_date DATE NOT NULL,
 status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','cancelled')),
 executed_on DATE,
 completed_at TIMESTAMPTZ,
 completed_by UUID REFERENCES users(id) ON DELETE RESTRICT,
 comment TEXT NOT NULL DEFAULT '',
 cancellation_reason TEXT NOT NULL DEFAULT '',
 series_id UUID REFERENCES document_deadline_series(id) ON DELETE RESTRICT,
 iteration INTEGER NOT NULL DEFAULT 0,
 planned_due_date DATE,
 version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK ((status='completed' AND executed_on IS NOT NULL AND completed_at IS NOT NULL AND completed_by IS NOT NULL AND cancellation_reason='')
     OR (status='active' AND executed_on IS NULL AND completed_at IS NULL AND completed_by IS NULL AND cancellation_reason='')
     OR (status='cancelled' AND executed_on IS NULL AND completed_at IS NULL AND completed_by IS NULL AND length(btrim(cancellation_reason))>0)),
 CHECK ((series_id IS NULL AND iteration=0 AND planned_due_date IS NULL)
     OR (series_id IS NOT NULL AND iteration>0 AND planned_due_date IS NOT NULL)),
 UNIQUE(series_id,iteration),
 UNIQUE(id,series_id,document_id,iteration)
);
ALTER TABLE document_deadline_series ADD CONSTRAINT deadline_series_current_control
 FOREIGN KEY(current_control_id,id,document_id,current_iteration)
 REFERENCES document_deadline_controls(id,series_id,document_id,iteration) DEFERRABLE INITIALLY DEFERRED;
CREATE UNIQUE INDEX deadline_series_one_active_iteration ON document_deadline_controls(series_id) WHERE status='active';
CREATE INDEX deadline_controls_document ON document_deadline_controls(document_id);
CREATE INDEX deadline_controls_author ON document_deadline_controls(created_by);
CREATE INDEX deadline_controls_executor ON document_deadline_controls(executor_id);
CREATE INDEX deadline_controls_co_executors ON document_deadline_controls USING gin(co_executor_ids);
CREATE INDEX deadline_controls_active_due ON document_deadline_controls(due_date) WHERE status='active';
CREATE INDEX deadline_series_document ON document_deadline_series(document_id);
CREATE INDEX deadline_series_author ON document_deadline_series(created_by);

CREATE INDEX deadline_series_executor ON document_deadline_series(executor_id);
CREATE INDEX deadline_series_co_executors ON document_deadline_series USING gin(co_executor_ids);
CREATE FUNCTION deadline_executor_set_valid(main UUID, co UUID[]) RETURNS BOOLEAN
LANGUAGE SQL IMMUTABLE AS $$
 SELECT NOT main=ANY(co) AND array_position(co,NULL) IS NULL
    AND cardinality(co)=(SELECT count(DISTINCT id) FROM unnest(co) AS id);
$$;
ALTER TABLE document_deadline_controls ADD CHECK(deadline_executor_set_valid(executor_id,co_executor_ids));
ALTER TABLE document_deadline_series ADD CHECK(deadline_executor_set_valid(executor_id,co_executor_ids));
ALTER TABLE document_deadline_controls ADD CHECK(executed_on IS NULL OR executed_on <= (completed_at AT TIME ZONE 'Asia/Yekaterinburg')::date);

-- Preserve referential integrity for co-executor arrays as well as the primary FK.
CREATE FUNCTION verify_deadline_co_executors() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 PERFORM id FROM deadline_executors WHERE id=ANY(NEW.co_executor_ids) FOR KEY SHARE;
 IF EXISTS(SELECT 1 FROM unnest(NEW.co_executor_ids) AS co(executor_id) WHERE NOT EXISTS(SELECT 1 FROM deadline_executors e WHERE e.id=co.executor_id)) THEN
  RAISE EXCEPTION 'co-executor does not exist' USING ERRCODE='23503';
 END IF;
 RETURN NEW;
END;
$$;
CREATE CONSTRAINT TRIGGER deadline_control_co_executor_refs AFTER INSERT OR UPDATE OF co_executor_ids ON document_deadline_controls
 DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION verify_deadline_co_executors();
CREATE CONSTRAINT TRIGGER deadline_series_co_executor_refs AFTER INSERT OR UPDATE OF co_executor_ids ON document_deadline_series
 DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION verify_deadline_co_executors();
CREATE FUNCTION restrict_deadline_co_executor_deletion() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM document_deadline_controls WHERE OLD.id=ANY(co_executor_ids))
    OR EXISTS(SELECT 1 FROM document_deadline_series WHERE OLD.id=ANY(co_executor_ids)) THEN
  RAISE EXCEPTION 'executor is referenced by deadline controls' USING ERRCODE='23503';
 END IF;
 RETURN OLD;
END;
$$;
CREATE TRIGGER deadline_co_executor_deletion BEFORE DELETE ON deadline_executors
 FOR EACH ROW EXECUTE FUNCTION restrict_deadline_co_executor_deletion();

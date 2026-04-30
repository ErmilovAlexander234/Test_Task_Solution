-- Recurrence rules table.
-- The rule's type-specific parameters are stored as a JSON payload,
-- which avoids a wide nullable-columns schema and keeps the table stable
-- as new recurrence types are added in the future.
CREATE TABLE IF NOT EXISTS recurrence_rules (
                                                id      BIGSERIAL PRIMARY KEY,
                                                type    TEXT      NOT NULL,
                                                payload JSONB     NOT NULL DEFAULT '{}'
);

-- Add recurrence metadata to tasks.
-- recurrence_rule_id links an instance back to its rule (nullable for one-off tasks).
-- scheduled_date records the calendar date this instance is intended for.
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS recurrence_rule_id BIGINT REFERENCES recurrence_rules(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS scheduled_date      DATE;

CREATE INDEX IF NOT EXISTS idx_tasks_recurrence_rule_id ON tasks (recurrence_rule_id);
CREATE INDEX IF NOT EXISTS idx_tasks_scheduled_date     ON tasks (scheduled_date);

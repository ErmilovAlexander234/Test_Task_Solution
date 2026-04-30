ALTER TABLE tasks
DROP COLUMN IF EXISTS scheduled_date,
    DROP COLUMN IF EXISTS recurrence_rule_id;

DROP TABLE IF EXISTS recurrence_rules;

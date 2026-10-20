-- A step's due time is fixed when it opens; overdue_notified_at makes the notice once only.
ALTER TABLE approval.instance_steps
    ADD COLUMN due_hours int CHECK (due_hours BETWEEN 1 AND 720),
    ADD COLUMN due_at timestamptz,
    ADD COLUMN overdue_notified_at timestamptz;
-- The overdue job scans only open steps that have a due time and were not yet told.
CREATE INDEX instance_steps_due_idx ON approval.instance_steps (due_at) WHERE status = 'open' AND overdue_notified_at IS NULL;

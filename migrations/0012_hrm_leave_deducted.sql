-- Days taken from the balance when the request was approved, so a cancel gives back exactly
-- that, whatever the kind of leave says by then.
ALTER TABLE hrm.leave_requests ADD COLUMN deducted numeric(4, 1) NOT NULL DEFAULT 0;

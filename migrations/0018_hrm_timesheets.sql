-- A timesheet is a document (see hrm.leave_requests for the deferred FK): the days worked
-- in one org unit over one payroll period. Cancelling one clears active, so the unit may
-- have a new timesheet for that period.
CREATE TABLE hrm.timesheets (
    id           bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    org_unit_id  bigint NOT NULL REFERENCES iam.org_units,
    period_start date NOT NULL,
    period_end   date NOT NULL CHECK (period_end >= period_start),
    active       boolean NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX timesheets_org_unit_period_key ON hrm.timesheets (org_unit_id, period_start) WHERE active;

-- One employee, one day: a full or half day actually worked. A blank cell has no row.
CREATE TABLE hrm.timesheet_lines (
    timesheet_id bigint NOT NULL REFERENCES hrm.timesheets ON DELETE CASCADE,
    employee_id  bigint NOT NULL REFERENCES hrm.employees,
    date         date NOT NULL,
    days         numeric(2, 1) NOT NULL CHECK (days IN (0.5, 1)),
    PRIMARY KEY (timesheet_id, employee_id, date)
);

CREATE INDEX timesheet_lines_employee_id_idx ON hrm.timesheet_lines (employee_id);

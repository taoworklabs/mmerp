-- An overtime request is a document (see hrm.leave_requests for the deferred FK): the hours
-- of one day, split into day and night shift, in half-hour steps.
CREATE TABLE hrm.overtime_requests (
    id          bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    employee_id bigint NOT NULL REFERENCES hrm.employees,
    date        date NOT NULL,
    day_kind    text NOT NULL CHECK (day_kind IN ('weekday', 'weekly_off', 'holiday')),
    day_hours   numeric(4, 1) NOT NULL,
    night_hours numeric(4, 1) NOT NULL,
    reason      text,
    CONSTRAINT overtime_requests_hours_check CHECK (
        day_hours >= 0 AND night_hours >= 0 AND day_hours + night_hours > 0 AND day_hours + night_hours <= 24
        AND day_hours * 2 = trunc(day_hours * 2) AND night_hours * 2 = trunc(night_hours * 2))
);

CREATE INDEX overtime_requests_employee_id_idx ON hrm.overtime_requests (employee_id);

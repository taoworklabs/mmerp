-- The months the accrual job has decided for an employee, with the days each one credited
-- (0 for a month that did not count), so no month is credited twice.
CREATE TABLE hrm.leave_accruals (
    employee_id bigint NOT NULL REFERENCES hrm.employees,
    year        int NOT NULL,
    month       int NOT NULL CHECK (month BETWEEN 1 AND 12),
    days        numeric(4, 1) NOT NULL CHECK (days >= 0),
    PRIMARY KEY (employee_id, year, month)
);

-- Days carried over from last year still unspent, part of days, and when they lapse. A
-- negative days is leave_balances_days_check's to report, so this check stays out of it.
ALTER TABLE hrm.leave_balances
    ADD COLUMN carried numeric(5, 1) NOT NULL DEFAULT 0,
    ADD COLUMN carried_until date,
    ADD CONSTRAINT leave_balances_carried_check CHECK (carried BETWEEN 0 AND greatest(days, 0));

-- Half a day on the first or last working date; computed rows take their days from the
-- calendar, older rows keep the count typed when they were saved. deducted_carried is the
-- carried part of deducted.
ALTER TABLE hrm.leave_requests
    ADD COLUMN start_half boolean NOT NULL DEFAULT false,
    ADD COLUMN end_half boolean NOT NULL DEFAULT false,
    ADD COLUMN computed boolean NOT NULL DEFAULT false,
    ADD COLUMN deducted_carried numeric(4, 1) NOT NULL DEFAULT 0;

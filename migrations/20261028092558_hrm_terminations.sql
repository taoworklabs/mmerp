-- A termination is a document (see hrm.leave_requests for the deferred FK); its date is the
-- last day employed. previous_date is the employee's termination date it replaced on posting,
-- restored on cancelling. No amount is stored: the payroll computes the allowance.
CREATE TABLE hrm.terminations (
    id               bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    employee_id      bigint NOT NULL REFERENCES hrm.employees,
    reason           text NOT NULL,
    decision_date    date,
    pension_eligible boolean NOT NULL DEFAULT false,
    insured_months   integer NOT NULL CHECK (insured_months >= 0),
    paid_months      integer NOT NULL DEFAULT 0 CHECK (paid_months >= 0),
    handover         text,
    note             text,
    previous_date    date
);

CREATE INDEX terminations_employee_id_idx ON hrm.terminations (employee_id);

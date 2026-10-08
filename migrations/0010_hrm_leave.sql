-- Kinds of leave, entered by the customer; only deducting kinds touch the leave balance.
CREATE TABLE hrm.leave_types (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name            text NOT NULL,
    deducts_balance boolean NOT NULL,
    active          boolean NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX leave_types_name_key ON hrm.leave_types (lower(name));

-- A leave request is a document: its id, status, version and date live in record.documents,
-- written first in the same transaction, hence the deferred FK.
CREATE TABLE hrm.leave_requests (
    id            bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    employee_id   bigint NOT NULL REFERENCES hrm.employees,
    leave_type_id bigint NOT NULL REFERENCES hrm.leave_types,
    start_date    date NOT NULL,
    end_date      date NOT NULL,
    days          numeric(4, 1) NOT NULL,
    reason        text,
    CONSTRAINT leave_requests_days_check CHECK (days > 0 AND days * 2 = trunc(days * 2) AND days <= end_date - start_date + 1),
    CONSTRAINT leave_requests_dates_check CHECK (end_date >= start_date AND date_trunc('year', start_date) = date_trunc('year', end_date))
);

CREATE INDEX leave_requests_employee_id_idx ON hrm.leave_requests (employee_id);

-- Remaining leave days of an employee in a year.
CREATE TABLE hrm.leave_balances (
    employee_id bigint NOT NULL REFERENCES hrm.employees,
    year        int NOT NULL,
    days        numeric(5, 1) NOT NULL CONSTRAINT leave_balances_days_check CHECK (days >= 0),
    PRIMARY KEY (employee_id, year)
);

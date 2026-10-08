CREATE SCHEMA hrm;

-- Sensitive columns are encrypted by the service (AAD = schema.table.column).
CREATE TABLE hrm.employees (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code                text NOT NULL,
    full_name           text NOT NULL,
    date_of_birth       date,
    gender              text CHECK (gender IN ('male', 'female', 'other')),
    phone               text,
    email               text,
    address             text,
    org_unit_id         bigint NOT NULL REFERENCES iam.org_units,
    manager_id          bigint REFERENCES hrm.employees,
    user_id             bigint UNIQUE REFERENCES iam.users,
    hire_date           date NOT NULL,
    termination_date    date CHECK (termination_date >= hire_date),
    national_id         bytea,
    social_insurance_no bytea,
    tax_code            bytea,
    bank_account        bytea,
    CHECK (manager_id <> id)
);

CREATE UNIQUE INDEX employees_code_key ON hrm.employees (lower(code));
CREATE INDEX employees_org_unit_id_idx ON hrm.employees (org_unit_id);

-- Every dependent field is sensitive, so the whole record is one encrypted JSON.
CREATE TABLE hrm.dependents (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    employee_id bigint NOT NULL REFERENCES hrm.employees ON DELETE CASCADE,
    data        bytea NOT NULL
);

CREATE INDEX dependents_employee_id_idx ON hrm.dependents (employee_id);

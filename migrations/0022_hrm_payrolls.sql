-- A payroll is a document (see hrm.leave_requests for the deferred FK): the pay of a legal
-- entity's employees over one period. inputs_hash is the sha256 of the sources it was computed
-- from; totals sums each department, never one person.
CREATE TABLE hrm.payrolls (
    id           bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    period_start date NOT NULL,
    period_end   date NOT NULL CHECK (period_end >= period_start),
    computed_at  timestamptz,
    inputs_hash  bytea,
    totals       jsonb NOT NULL DEFAULT '[]'
);

-- One employee's pay: the inputs read and every amount, one encrypted JSON (AAD hrm.payroll_lines.data).
CREATE TABLE hrm.payroll_lines (
    payroll_id  bigint NOT NULL REFERENCES hrm.payrolls ON DELETE CASCADE,
    employee_id bigint NOT NULL REFERENCES hrm.employees,
    org_unit_id bigint NOT NULL REFERENCES iam.org_units,
    data        bytea NOT NULL,
    PRIMARY KEY (payroll_id, employee_id)
);

-- The posted source documents a payroll was computed from, at their versions.
CREATE TABLE hrm.payroll_sources (
    payroll_id bigint NOT NULL REFERENCES hrm.payrolls ON DELETE CASCADE,
    doc_type   text NOT NULL,
    doc_id     bigint NOT NULL,
    version    int NOT NULL,
    PRIMARY KEY (payroll_id, doc_type, doc_id)
);

-- Back pay (+) or recovery (-) of a closed period, entered on a draft payroll; amount and
-- reason are one encrypted JSON (AAD hrm.payroll_adjustments.data).
CREATE TABLE hrm.payroll_adjustments (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    payroll_id    bigint NOT NULL REFERENCES hrm.payrolls ON DELETE CASCADE,
    employee_id   bigint NOT NULL REFERENCES hrm.employees,
    source_period date NOT NULL,
    data          bytea NOT NULL
);

CREATE INDEX payroll_adjustments_payroll_id_idx ON hrm.payroll_adjustments (payroll_id);

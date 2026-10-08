-- One row per legal entity, only to lock: source documents take it FOR SHARE, posting a payroll
-- FOR UPDATE. Rows are created on first use.
CREATE TABLE hrm.payroll_locks (
    legal_entity_id bigint PRIMARY KEY REFERENCES iam.org_units
);

-- Payroll periods of a legal entity; a period is closed while posted_payroll_id is set.
CREATE TABLE hrm.payroll_periods (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    legal_entity_id   bigint NOT NULL REFERENCES iam.org_units,
    period_start      date NOT NULL,
    period_end        date NOT NULL CHECK (period_end >= period_start),
    posted_payroll_id bigint REFERENCES record.documents,
    UNIQUE (legal_entity_id, period_start)
);

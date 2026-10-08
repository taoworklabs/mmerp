CREATE SCHEMA posting;

-- Posting lines: what a posted document means economically, captured once. Amounts are signed
-- (negative reduces) and never carry one person's sensitive amounts. Voiding keeps the lines.
CREATE TABLE posting.lines (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type        text NOT NULL,
    doc_id          bigint NOT NULL,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    date            date NOT NULL,
    kind            text NOT NULL,
    org_unit_id     bigint NOT NULL REFERENCES iam.org_units,
    amount          bigint NOT NULL,
    voided_at       timestamptz
);

CREATE INDEX lines_doc_idx ON posting.lines (doc_type, doc_id);

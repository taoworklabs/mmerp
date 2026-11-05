-- Which account a posting kind is booked to, and on which side.
CREATE TABLE accounting.rules (
    kind       text PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES accounting.accounts,
    side       text NOT NULL CHECK (side IN ('debit', 'credit'))
);

-- The day a legal entity starts booking in accounting.
CREATE TABLE accounting.starts (
    legal_entity_id bigint PRIMARY KEY REFERENCES iam.org_units,
    start_date      date NOT NULL
);

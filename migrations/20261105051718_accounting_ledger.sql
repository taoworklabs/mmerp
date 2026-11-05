-- Journal entries: written once, never edited or deleted; a reversal points at the entry it undoes.
CREATE TABLE accounting.entries (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    date            date NOT NULL,
    doc_type        text NOT NULL,
    doc_id          bigint NOT NULL,
    recording_id    bigint NULL REFERENCES posting.recordings,
    reverses        bigint NULL UNIQUE REFERENCES accounting.entries,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX entries_doc_idx ON accounting.entries (doc_type, doc_id);
CREATE INDEX entries_legal_entity_date_idx ON accounting.entries (legal_entity_id, date);

CREATE TABLE accounting.entry_lines (
    entry_id    bigint NOT NULL REFERENCES accounting.entries,
    line_no     int NOT NULL,
    account_id  bigint NOT NULL REFERENCES accounting.accounts,
    org_unit_id bigint NULL REFERENCES iam.org_units,
    kind        text NULL,
    description text NULL,
    debit       bigint NOT NULL CHECK (debit >= 0),
    credit      bigint NOT NULL CHECK (credit >= 0),
    CHECK ((debit = 0) <> (credit = 0)),
    PRIMARY KEY (entry_id, line_no)
);

CREATE INDEX entry_lines_account_idx ON accounting.entry_lines (account_id);

-- One row per source document the ledger has looked at: which recording it shows, or why none.
CREATE TABLE accounting.sources (
    doc_type            text NOT NULL,
    doc_id              bigint NOT NULL,
    legal_entity_id     bigint NOT NULL,
    date                date NOT NULL,
    posted_recording_id bigint NULL,
    problem             text NULL CHECK (problem IN ('unmapped', 'unbalanced', 'too_large')),
    missing_kinds       text[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (doc_type, doc_id)
);

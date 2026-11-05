-- Manual journal entries: a document an accountant enters by hand. Posting writes the same
-- accounting.entries as an automatic one.
CREATE TABLE accounting.journal_entries (
    id          bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    description text NOT NULL CHECK (char_length(description) BETWEEN 1 AND 500)
);

CREATE TABLE accounting.journal_entry_lines (
    journal_entry_id bigint NOT NULL REFERENCES accounting.journal_entries,
    line_no          int NOT NULL,
    account_id       bigint NOT NULL REFERENCES accounting.accounts,
    org_unit_id      bigint NULL REFERENCES iam.org_units,
    description      text NULL CHECK (description IS NULL OR char_length(description) BETWEEN 1 AND 500),
    debit            bigint NOT NULL CHECK (debit BETWEEN 0 AND 1000000000000000),
    credit           bigint NOT NULL CHECK (credit BETWEEN 0 AND 1000000000000000),
    CHECK ((debit = 0) <> (credit = 0)),
    PRIMARY KEY (journal_entry_id, line_no)
);

CREATE INDEX journal_entry_lines_account_idx ON accounting.journal_entry_lines (account_id);

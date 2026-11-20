-- Closing entries: documents whose ledger entry moves revenue and expense balances to the
-- result account, then the result to retained earnings.
CREATE TABLE accounting.closings (
    id          bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    description text NOT NULL CHECK (char_length(description) BETWEEN 1 AND 500)
);

-- The two accounts a closing posts to; at most one row.
CREATE TABLE accounting.closing_accounts (
    singleton           boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    result_account_id   bigint NOT NULL CONSTRAINT closing_accounts_result_fkey REFERENCES accounting.accounts,
    retained_account_id bigint NOT NULL CONSTRAINT closing_accounts_retained_fkey REFERENCES accounting.accounts
);

CREATE SCHEMA accounting;

CREATE TABLE accounting.accounts (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code      text NOT NULL UNIQUE CHECK (code ~ '^[0-9A-Za-z.]{1,20}$'),
    name      text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    type      text NOT NULL CHECK (type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
    parent_id bigint NULL REFERENCES accounting.accounts,
    active    boolean NOT NULL DEFAULT true
);

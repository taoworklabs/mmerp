-- Statement forms: the rows of the balance sheet and the income statement, each with the
-- formula that computes it from the ledger.
CREATE TABLE accounting.statement_rows (
    form    text NOT NULL CHECK (form IN ('balance_sheet', 'income_statement')),
    line_no int NOT NULL CHECK (line_no > 0),
    code    text NOT NULL CHECK (code ~ '^[0-9A-Za-z.]{1,10}$'),
    label   text NOT NULL CHECK (char_length(label) BETWEEN 1 AND 300),
    formula text NOT NULL CHECK (char_length(formula) <= 1000),
    bold    boolean NOT NULL DEFAULT false,
    PRIMARY KEY (form, line_no),
    UNIQUE (form, code)
);

CREATE SCHEMA setting;

-- A missing key means the default declared in code.
CREATE TABLE setting.values (
    key   text PRIMARY KEY,
    value text NOT NULL
);

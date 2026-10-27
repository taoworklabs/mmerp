-- One row: the tenant's password rules and sign-in lockout.
CREATE TABLE iam.login_policy (
    min_length   int NOT NULL DEFAULT 8 CHECK (min_length BETWEEN 8 AND 64),
    upper        boolean NOT NULL DEFAULT false,
    lower        boolean NOT NULL DEFAULT false,
    digit        boolean NOT NULL DEFAULT false,
    symbol       boolean NOT NULL DEFAULT false,
    max_failures int NOT NULL DEFAULT 5 CHECK (max_failures BETWEEN 3 AND 20),
    lock_minutes int NOT NULL DEFAULT 15 CHECK (lock_minutes BETWEEN 1 AND 1440)
);

INSERT INTO iam.login_policy DEFAULT VALUES;

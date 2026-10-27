-- Consecutive wrong passwords per typed login (lowercased), known or not.
CREATE TABLE iam.login_failures (
    login          text PRIMARY KEY,
    failures       int NOT NULL,
    last_failed_at timestamptz NOT NULL,
    locked_until   timestamptz
);

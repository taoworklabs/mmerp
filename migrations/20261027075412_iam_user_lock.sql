-- NULL means the account is active.
ALTER TABLE iam.users ADD COLUMN locked_at timestamptz;

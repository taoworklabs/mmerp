-- The one account created at startup; it cannot be locked or lose core.admin.
ALTER TABLE iam.users ADD COLUMN is_root boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX users_one_root ON iam.users (is_root) WHERE is_root;

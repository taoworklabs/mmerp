-- Who sent a pending document; only they withdraw it.
ALTER TABLE record.documents ADD COLUMN submitted_by bigint REFERENCES iam.users;

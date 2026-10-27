-- id is the handle a user sees when ending a session; token_hash stays the key.
ALTER TABLE iam.sessions ADD COLUMN id bigint GENERATED ALWAYS AS IDENTITY UNIQUE;
ALTER TABLE iam.sessions ADD COLUMN user_agent text;

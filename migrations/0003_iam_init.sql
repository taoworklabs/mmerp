CREATE SCHEMA iam;

CREATE TABLE iam.users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    login         text NOT NULL,
    name          text NOT NULL,
    password_hash text NOT NULL,
    locale        text CHECK (locale IN ('vi', 'en')),
    authz_version bigint NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_login_key ON iam.users (lower(login));

CREATE TABLE iam.sessions (
    token_hash bytea PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES iam.users ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON iam.sessions (user_id);

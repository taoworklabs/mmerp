CREATE SCHEMA dataio;

-- Uploaded import files and temporary export files; contents live in FILES_DIR under the same id,
-- written before this row. dataio.cleanup removes both once expired.
CREATE TABLE dataio.files (
    id         text PRIMARY KEY,
    name       text NOT NULL,
    owner_id   bigint NOT NULL REFERENCES iam.users,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX files_expires_at_idx ON dataio.files (expires_at);

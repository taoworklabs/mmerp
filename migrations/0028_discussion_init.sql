CREATE SCHEMA discussion;

-- Comments on a record of any type, named by (doc_type, doc_id) without a foreign key.
-- Plain text, never edited or deleted, except with a deleted draft.
CREATE TABLE discussion.comments (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type   text NOT NULL,
    doc_id     bigint NOT NULL,
    author_id  bigint NOT NULL REFERENCES iam.users,
    body       text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 4000),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX comments_doc_idx ON discussion.comments (doc_type, doc_id);

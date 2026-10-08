CREATE SCHEMA attachment;

-- Files attached to a record of any type, named by (doc_type, doc_id) without a foreign key.
-- Contents live in the attachment directory under file_id, written before this row;
-- attachment.cleanup removes contents no row names any more.
CREATE TABLE attachment.files (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type     text NOT NULL,
    doc_id       bigint NOT NULL,
    file_id      text NOT NULL UNIQUE,
    name         text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255),
    size         bigint NOT NULL CHECK (size BETWEEN 1 AND 20971520),
    content_type text NOT NULL CHECK (content_type IN (
        'application/pdf', 'image/jpeg', 'image/png',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')),
    uploaded_by  bigint NOT NULL REFERENCES iam.users,
    uploaded_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX files_doc_idx ON attachment.files (doc_type, doc_id);

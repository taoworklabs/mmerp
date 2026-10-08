CREATE SCHEMA notification;

-- What a user is told of: a record (doc_type, doc_id, no foreign key) or one of their jobs.
-- No text: the reader builds it from kind, so nothing of the record is copied here.
CREATE TABLE notification.notifications (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES iam.users,
    kind       text NOT NULL CHECK (kind IN ('approval_requested', 'approval_approved', 'approval_rejected', 'mentioned', 'job_completed', 'job_failed')),
    doc_type   text,
    doc_id     bigint,
    job_id     bigint,
    actor_id   bigint REFERENCES iam.users,
    created_at timestamptz NOT NULL DEFAULT now(),
    read_at    timestamptz,
    CHECK ((doc_type IS NOT NULL AND doc_id IS NOT NULL AND job_id IS NULL)
        OR (doc_type IS NULL AND doc_id IS NULL AND job_id IS NOT NULL))
);

CREATE INDEX notifications_user_idx ON notification.notifications (user_id, id DESC);
CREATE INDEX notifications_unread_idx ON notification.notifications (user_id) WHERE read_at IS NULL;

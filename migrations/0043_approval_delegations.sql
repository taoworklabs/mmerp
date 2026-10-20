CREATE TABLE approval.delegations (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     bigint NOT NULL REFERENCES iam.users,
    delegate_id bigint NOT NULL REFERENCES iam.users,
    starts_on   date NOT NULL,
    ends_on     date NOT NULL,
    created_by  bigint NOT NULL REFERENCES iam.users,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (delegate_id <> user_id),
    CHECK (ends_on >= starts_on)
);
CREATE INDEX delegations_delegate_idx ON approval.delegations (delegate_id, starts_on, ends_on);
CREATE INDEX delegations_user_idx ON approval.delegations (user_id, starts_on, ends_on);

ALTER TABLE approval.step_votes ADD COLUMN cast_by bigint REFERENCES iam.users;
-- One vote per person per step, whether in person or for someone else.
CREATE UNIQUE INDEX step_votes_one_per_person_idx ON approval.step_votes (instance_id, position, (coalesce(cast_by, user_id)));

CREATE SCHEMA approval;

-- At most one rule per document type; steps is a JSON array edited as a whole.
CREATE TABLE approval.rules (
    doc_type         text PRIMARY KEY,
    steps            jsonb NOT NULL,
    max_levels       int NOT NULL CHECK (max_levels BETWEEN 1 AND 10),
    fallback_product text NOT NULL,
    fallback_role    text NOT NULL
);

-- One submission of a document. It copies what it needs from the rule, so editing
-- the rule never changes an open instance. No FK to the document: deleting a draft keeps its history.
CREATE TABLE approval.instances (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type         text NOT NULL,
    doc_id           bigint NOT NULL,
    version          int NOT NULL,
    submitted_by     bigint NOT NULL REFERENCES iam.users,
    submitted_at     timestamptz NOT NULL DEFAULT now(),
    status           text NOT NULL CHECK (status IN ('open', 'approved', 'rejected', 'withdrawn', 'stale')),
    current_step     int NOT NULL,
    max_levels       int NOT NULL,
    fallback_product text NOT NULL,
    fallback_role    text NOT NULL
);

CREATE INDEX instances_doc_idx ON approval.instances (doc_type, doc_id);

-- The steps whose condition held at submission. approvers is fixed when the step starts.
CREATE TABLE approval.instance_steps (
    instance_id bigint NOT NULL REFERENCES approval.instances,
    position    int NOT NULL,
    approver    jsonb NOT NULL,
    approvers   bigint[],
    fallback    boolean NOT NULL DEFAULT false,
    decided_by  bigint REFERENCES iam.users,
    decision    text CHECK (decision IN ('approved', 'rejected')),
    reason      text,
    decided_at  timestamptz,
    PRIMARY KEY (instance_id, position)
);

CREATE INDEX instance_steps_approvers_idx ON approval.instance_steps USING gin (approvers) WHERE decision IS NULL;

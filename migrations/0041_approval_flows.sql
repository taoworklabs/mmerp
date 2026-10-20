-- Rules: steps (a chain) become a flow; a step with a condition becomes a choose-one
-- split with an empty otherwise, which skips it exactly as before.
ALTER TABLE approval.rules ADD COLUMN flow jsonb;
UPDATE approval.rules r SET flow = (
    SELECT jsonb_agg(CASE WHEN s->'condition' IS NULL OR s->'condition' = 'null'::jsonb THEN jsonb_build_object('step', n.def)
        ELSE jsonb_build_object('split', jsonb_build_object('mode', 'one', 'otherwise', '[]'::jsonb,
            'branches', jsonb_build_array(jsonb_build_object('condition', s->'condition', 'body', jsonb_build_array(jsonb_build_object('step', n.def))))))
        END ORDER BY i)
    FROM jsonb_array_elements(r.steps) WITH ORDINALITY e(s, i)
    CROSS JOIN LATERAL (SELECT jsonb_build_object('approvals', 1, 'rejections', 1, 'approver',
        CASE WHEN s->'approver'->>'kind' = 'user' THEN jsonb_build_object('kind', 'user', 'user_ids', jsonb_build_array(s->'approver'->'user_id'))
             ELSE s->'approver' END) AS def) n);
ALTER TABLE approval.rules ALTER COLUMN flow SET NOT NULL, DROP COLUMN steps;

-- Instances: the resolved plan replaces current_step; old instances were a plain sequence.
ALTER TABLE approval.instances ADD COLUMN plan jsonb;
UPDATE approval.instances i SET plan = jsonb_build_object('seq',
    (SELECT jsonb_agg(jsonb_build_object('step', s.position) ORDER BY s.position) FROM approval.instance_steps s WHERE s.instance_id = i.id));

-- Steps: counts and a status; each vote moves to its own row.
ALTER TABLE approval.instance_steps
    ADD COLUMN approvals int NOT NULL DEFAULT 1 CHECK (approvals BETWEEN 1 AND 20),
    ADD COLUMN rejections int NOT NULL DEFAULT 1 CHECK (rejections BETWEEN 1 AND 20),
    ADD COLUMN status text;
UPDATE approval.instance_steps s SET
    status = CASE WHEN s.decision IS NOT NULL THEN s.decision
                  WHEN i.status = 'open' AND s.position = i.current_step THEN 'open'
                  WHEN i.status = 'open' THEN 'waiting'
                  ELSE 'cancelled' END,
    approver = CASE WHEN s.approver->>'kind' = 'user'
                    THEN jsonb_build_object('kind', 'user', 'user_ids', jsonb_build_array(s.approver->'user_id')) ELSE s.approver END
FROM approval.instances i WHERE i.id = s.instance_id;
ALTER TABLE approval.instance_steps ALTER COLUMN status SET NOT NULL,
    ADD CONSTRAINT instance_steps_status_check CHECK (status IN ('waiting', 'open', 'approved', 'rejected', 'cancelled'));

CREATE TABLE approval.step_votes (
    instance_id bigint NOT NULL,
    position    int NOT NULL,
    user_id     bigint NOT NULL REFERENCES iam.users,
    decision    text NOT NULL CHECK (decision IN ('approved', 'rejected')),
    reason      text,
    decided_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (instance_id, position, user_id),
    FOREIGN KEY (instance_id, position) REFERENCES approval.instance_steps
);
INSERT INTO approval.step_votes (instance_id, position, user_id, decision, reason, decided_at)
SELECT instance_id, position, decided_by, decision, reason, decided_at FROM approval.instance_steps WHERE decided_by IS NOT NULL;

DROP INDEX approval.instance_steps_approvers_idx;
ALTER TABLE approval.instance_steps DROP COLUMN decided_by, DROP COLUMN decision, DROP COLUMN reason, DROP COLUMN decided_at;
ALTER TABLE approval.instances ALTER COLUMN plan SET NOT NULL, DROP COLUMN current_step;
CREATE INDEX instance_steps_open_idx ON approval.instance_steps USING gin (approvers) WHERE status = 'open';

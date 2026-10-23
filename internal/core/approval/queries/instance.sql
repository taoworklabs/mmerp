-- name: InsertInstance :one
INSERT INTO approval.instances (doc_type, doc_id, version, submitted_by, status, plan, max_levels, fallback_product, fallback_role, action, params)
VALUES ($1, $2, $3, $4, 'open', $5, $6, $7, $8, $9, $10) RETURNING id;

-- name: HeldBy :one
-- The open request holding a document for an action other than post.
SELECT * FROM approval.instances WHERE doc_type = $1 AND doc_id = $2 AND status = 'open' AND action <> 'post';

-- name: HeldInPeriod :many
SELECT d.doc_type, d.id, d.number, d.date
FROM approval.instances i JOIN record.documents d ON d.id = i.doc_id AND d.doc_type = i.doc_type
WHERE i.status = 'open' AND i.action <> 'post' AND d.legal_entity_id = $1 AND d.date <= $2
ORDER BY d.date, d.id;

-- name: InsertStep :exec
INSERT INTO approval.instance_steps (instance_id, position, approver, approvals, rejections, due_hours, status)
VALUES ($1, $2, $3, $4, $5, $6, 'waiting');

-- name: StartStep :exec
UPDATE approval.instance_steps SET approvers = @approvers, fallback = @fallback, status = 'open',
    due_at = now() + due_hours * interval '1 hour'
WHERE instance_id = @instance_id AND position = @position;

-- name: ClaimOverdue :one
-- Marks one overdue open step as told and returns it; SKIP LOCKED lets concurrent runs share the work.
UPDATE approval.instance_steps SET overdue_notified_at = now()
WHERE (instance_id, position) = (
    SELECT s.instance_id, s.position FROM approval.instance_steps s JOIN approval.instances i ON i.id = s.instance_id
    WHERE s.status = 'open' AND i.status = 'open' AND s.overdue_notified_at IS NULL AND s.due_at < now()
    ORDER BY s.due_at LIMIT 1 FOR UPDATE OF s SKIP LOCKED)
RETURNING instance_id, position, approvers;

-- name: SetApprovers :exec
UPDATE approval.instance_steps SET approvers = @approvers WHERE instance_id = @instance_id AND position = @position;

-- name: SetStepStatus :exec
UPDATE approval.instance_steps SET status = $3 WHERE instance_id = $1 AND position = $2;

-- name: CancelOpenSteps :exec
UPDATE approval.instance_steps SET status = 'cancelled' WHERE instance_id = $1 AND status IN ('open', 'waiting');

-- name: GetInstance :one
SELECT * FROM approval.instances WHERE id = $1;

-- name: LockInstance :one
SELECT * FROM approval.instances WHERE id = $1 FOR UPDATE;

-- name: GetStep :one
SELECT * FROM approval.instance_steps WHERE instance_id = $1 AND position = $2;

-- name: StepStatuses :many
SELECT position, status FROM approval.instance_steps WHERE instance_id = $1 ORDER BY position;

-- name: InsertVote :exec
INSERT INTO approval.step_votes (instance_id, position, user_id, cast_by, decision, reason) VALUES ($1, $2, $3, $4, $5, $6);

-- name: CastVotes :many
SELECT user_id, cast_by FROM approval.step_votes WHERE instance_id = $1 AND position = $2;

-- name: StepVotes :one
SELECT count(*) FILTER (WHERE decision = 'approved')::int AS approved,
       count(*) FILTER (WHERE decision = 'rejected')::int AS rejected
FROM approval.step_votes WHERE instance_id = $1 AND position = $2;

-- name: HasVoted :one
SELECT EXISTS (SELECT 1 FROM approval.step_votes WHERE instance_id = $1 AND position = $2 AND user_id = $3);

-- name: CloseInstance :exec
UPDATE approval.instances SET status = $2 WHERE id = $1 AND status = 'open';

-- name: Inbox :many
-- One row per instance: the lowest open step where @actor has their own pending seat,
-- or a pending seat delegated to them, and has cast nothing yet. An actor in two open
-- parallel steps of the same instance sees it once, not twice.
SELECT i.id, i.doc_type, i.doc_id, i.action, s.step, i.submitted_at, i.submitted_by, u.name AS submitted_by_name, d.number, d.date,
       b.id AS on_behalf_of, b.name AS on_behalf_of_name
FROM (
    SELECT DISTINCT ON (s.instance_id) s.instance_id, s.position AS step,
           CASE WHEN NOT s.own THEN s.delegated END AS on_behalf_of
    FROM (
        SELECT s.instance_id, s.position,
               s.approvers @> ARRAY[@actor::bigint] AND NOT EXISTS (
                   SELECT 1 FROM approval.step_votes v WHERE v.instance_id = s.instance_id AND v.position = s.position AND v.user_id = @actor) AS own,
               (SELECT p.id FROM unnest(s.approvers) WITH ORDINALITY AS p(id, n)
                JOIN approval.delegations g ON g.user_id = p.id AND g.delegate_id = @actor AND g.starts_on <= @today AND g.ends_on >= @today
                WHERE NOT EXISTS (SELECT 1 FROM approval.step_votes v WHERE v.instance_id = s.instance_id AND v.position = s.position AND v.user_id = p.id)
                ORDER BY p.n LIMIT 1) AS delegated
        FROM approval.instance_steps s
        WHERE s.status = 'open'
          -- Served by the open steps' approvers index.
          AND s.approvers && (ARRAY[@actor::bigint] || ARRAY(SELECT g.user_id FROM approval.delegations g
                                                            WHERE g.delegate_id = @actor AND g.starts_on <= @today AND g.ends_on >= @today))
          AND NOT EXISTS (SELECT 1 FROM approval.step_votes v
                          WHERE v.instance_id = s.instance_id AND v.position = s.position AND coalesce(v.cast_by, v.user_id) = @actor)
    ) s
    WHERE s.own OR s.delegated IS NOT NULL
    ORDER BY s.instance_id, s.position
) s
JOIN approval.instances i ON i.id = s.instance_id AND i.status = 'open'
JOIN record.documents d ON d.id = i.doc_id
JOIN iam.users u ON u.id = i.submitted_by
LEFT JOIN iam.users b ON b.id = s.on_behalf_of
ORDER BY i.submitted_at, i.id;

-- name: LatestInstance :one
SELECT i.*, u.name AS submitted_by_name FROM approval.instances i JOIN iam.users u ON u.id = i.submitted_by
WHERE i.doc_type = $1 AND i.doc_id = $2 ORDER BY i.id DESC LIMIT 1;

-- name: Steps :many
-- Each step with its voters: who may vote and how each voted, by name.
SELECT s.position, s.status, s.approvals, s.rejections, s.fallback, s.approvers, s.due_at, coalesce(s.due_at < now(), false)::boolean AS past_due,
       (SELECT coalesce(jsonb_agg(jsonb_build_object('id', a.id, 'name', a.name, 'decision', v.decision, 'reason', v.reason, 'decided_at', v.decided_at,
                                                     'cast_by_name', c.name)
                                  ORDER BY a.name, a.id), '[]')
        FROM iam.users a LEFT JOIN approval.step_votes v ON v.instance_id = s.instance_id AND v.position = s.position AND v.user_id = a.id
        LEFT JOIN iam.users c ON c.id = v.cast_by
        WHERE a.id = ANY(s.approvers))::jsonb AS voters
FROM approval.instance_steps s
WHERE s.instance_id = $1 ORDER BY s.position;

-- name: Within :one
-- Whether @unit is @ancestor or below it.
WITH RECURSIVE up AS (
    SELECT o.id, o.parent_id FROM iam.org_units o WHERE o.id = @unit
    UNION
    SELECT o.id, o.parent_id FROM iam.org_units o JOIN up ON o.id = up.parent_id
)
SELECT EXISTS (SELECT 1 FROM up WHERE up.id = @ancestor::bigint);

-- name: ListUsers :many
-- Who a rule may name as approver, or a delegation may name.
SELECT id, login, name FROM iam.users ORDER BY lower(login);

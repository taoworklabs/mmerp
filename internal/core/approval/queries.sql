-- name: GetRule :one
SELECT * FROM approval.rules WHERE doc_type = $1;

-- name: ListRules :many
SELECT * FROM approval.rules ORDER BY doc_type;

-- name: SaveRule :exec
INSERT INTO approval.rules (doc_type, steps, max_levels, fallback_product, fallback_role) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (doc_type) DO UPDATE SET steps = $2, max_levels = $3, fallback_product = $4, fallback_role = $5;

-- name: DeleteRule :exec
DELETE FROM approval.rules WHERE doc_type = $1;

-- name: InsertInstance :one
INSERT INTO approval.instances (doc_type, doc_id, version, submitted_by, status, current_step, max_levels, fallback_product, fallback_role)
VALUES ($1, $2, $3, $4, 'open', 1, $5, $6, $7) RETURNING id;

-- name: InsertStep :exec
INSERT INTO approval.instance_steps (instance_id, position, approver) VALUES ($1, $2, $3);

-- name: StartStep :exec
UPDATE approval.instance_steps SET approvers = @approvers, fallback = @fallback WHERE instance_id = @instance_id AND position = @position;

-- name: GetInstance :one
SELECT * FROM approval.instances WHERE id = $1;

-- name: LockInstance :one
SELECT * FROM approval.instances WHERE id = $1 FOR UPDATE;

-- name: GetStep :one
SELECT * FROM approval.instance_steps WHERE instance_id = $1 AND position = $2;

-- name: StepCount :one
SELECT count(*) FROM approval.instance_steps WHERE instance_id = $1;

-- name: Decide :exec
UPDATE approval.instance_steps SET decided_by = @decided_by, decision = @decision, reason = @reason, decided_at = now()
WHERE instance_id = @instance_id AND position = @position;

-- name: SetCurrentStep :exec
UPDATE approval.instances SET current_step = $2 WHERE id = $1;

-- name: CloseInstance :exec
UPDATE approval.instances SET status = $2 WHERE id = $1 AND status = 'open';

-- name: Inbox :many
-- Open instances waiting for @actor at their current step.
SELECT i.id, i.doc_type, i.doc_id, i.current_step, i.submitted_at, u.name AS submitted_by_name, d.number, d.date
FROM approval.instances i
JOIN approval.instance_steps s ON s.instance_id = i.id AND s.position = i.current_step
JOIN record.documents d ON d.id = i.doc_id
JOIN iam.users u ON u.id = i.submitted_by
WHERE i.status = 'open' AND s.decision IS NULL AND s.approvers @> ARRAY[@actor::bigint]
ORDER BY i.submitted_at, i.id;

-- name: LatestInstance :one
SELECT i.*, u.name AS submitted_by_name FROM approval.instances i JOIN iam.users u ON u.id = i.submitted_by
WHERE i.doc_type = $1 AND i.doc_id = $2 ORDER BY i.id DESC LIMIT 1;

-- name: Steps :many
SELECT s.position, s.approvers, s.fallback, s.decision, s.reason, s.decided_at, d.name AS decided_by_name,
       (SELECT coalesce(array_agg(a.name ORDER BY a.name), '{}')::text[] FROM iam.users a WHERE a.id = ANY(s.approvers)) AS approver_names
FROM approval.instance_steps s LEFT JOIN iam.users d ON d.id = s.decided_by
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
-- Who a rule may name as approver.
SELECT id, login, name FROM iam.users ORDER BY lower(login);

-- name: InsertDocument :one
INSERT INTO record.documents (doc_type, number, status, version, date, legal_entity_id, org_unit_id, amount, fields)
VALUES (@doc_type, @number, 'draft', 1, @date, @legal_entity_id, @org_unit_id, @amount, @fields)
RETURNING id;

-- name: GetDocument :one
SELECT * FROM record.documents WHERE id = $1 AND doc_type = $2;

-- name: LockDocument :one
SELECT * FROM record.documents WHERE id = $1 FOR UPDATE;

-- name: UpdateHeader :exec
UPDATE record.documents SET date = $2, org_unit_id = $3, amount = $4, fields = $5, version = version + 1
WHERE id = $1;

-- name: SetStatus :exec
-- Sending for approval keeps the version, so the approval instance can compare it.
UPDATE record.documents SET status = @status, approval_ticket = @approval_ticket, submitted_by = @submitted_by,
    version = version + @bump::int
WHERE id = @id;

-- name: DeleteDocument :exec
DELETE FROM record.documents WHERE id = $1;

-- name: EnsurePeriodLock :exec
INSERT INTO record.period_locks (legal_entity_id) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: SharePeriodLock :one
SELECT locked_until FROM record.period_locks WHERE legal_entity_id = $1 FOR SHARE;

-- name: LockPeriodLock :one
SELECT locked_until FROM record.period_locks WHERE legal_entity_id = $1 FOR UPDATE;

-- name: LockedUntil :many
-- Plain read for allowed_actions; no row means nothing is locked.
SELECT locked_until FROM record.period_locks WHERE legal_entity_id = $1;

-- name: SetLockedUntil :exec
UPDATE record.period_locks SET locked_until = $2 WHERE legal_entity_id = $1;

-- name: PendingInPeriod :many
SELECT id, doc_type, number, date FROM record.documents
WHERE legal_entity_id = $1 AND status = 'pending_approval' AND date <= $2
ORDER BY date, id;

-- name: PeriodLocks :many
SELECT u.id, u.name, p.locked_until FROM iam.org_units u
LEFT JOIN record.period_locks p ON p.legal_entity_id = u.id
WHERE u.kind = 'company' ORDER BY u.name, u.id;

-- name: MisplacedDocuments :one
-- Documents whose org unit now sits under another legal entity than the one stored.
WITH RECURSIVE walk AS (
    SELECT id, CASE WHEN kind = 'company' THEN id END AS company FROM iam.org_units WHERE parent_id IS NULL
    UNION ALL
    SELECT u.id, CASE WHEN u.kind = 'company' THEN u.id ELSE w.company END
    FROM iam.org_units u JOIN walk w ON u.parent_id = w.id
)
SELECT count(*) FROM record.documents d JOIN walk w ON w.id = d.org_unit_id
WHERE w.company IS DISTINCT FROM d.legal_entity_id;

-- name: History :many
SELECT l.id, l.at, u.name AS actor_name, l.action, l.data
FROM audit.log l LEFT JOIN iam.users u ON u.id = l.actor_id
WHERE l.doc_type = $1 AND l.doc_id = $2
ORDER BY l.at, l.id;

-- name: DocTypesInUse :many
SELECT t::text FROM unnest(@types::text[]) AS t
WHERE EXISTS (SELECT 1 FROM record.documents d WHERE d.doc_type = t);

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

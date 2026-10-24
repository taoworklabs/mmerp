-- name: InsertDocument :one
INSERT INTO record.documents (doc_type, number, status, version, date, legal_entity_id, org_unit_id, amount, fields,
    request_id, requested_by)
VALUES (@doc_type, @number, 'draft', 1, @date, @legal_entity_id, @org_unit_id, @amount, @fields,
    CAST(sqlc.narg(request_id)::text AS uuid), sqlc.narg(requested_by)::bigint)
RETURNING id;

-- name: GetDocument :one
SELECT * FROM record.documents WHERE id = $1 AND doc_type = $2;

-- name: LockDocument :one
SELECT * FROM record.documents WHERE id = $1 FOR UPDATE;

-- name: UpdateHeader :exec
UPDATE record.documents SET date = $2, org_unit_id = $3, amount = $4, fields = $5, version = version + 1
WHERE id = $1;

-- name: BumpVersion :exec
UPDATE record.documents SET version = version + 1 WHERE id = $1;

-- name: SetStatus :exec
-- Sending for approval keeps the version, so the approval instance can compare it.
UPDATE record.documents SET status = @status, approval_ticket = @approval_ticket, submitted_by = @submitted_by,
    version = version + @bump::int
WHERE id = @id;

-- name: DeleteDocument :exec
DELETE FROM record.documents WHERE id = $1;

-- name: History :many
SELECT l.id, l.at, u.name AS actor_name, l.action, l.data
FROM audit.log l LEFT JOIN iam.users u ON u.id = l.actor_id
WHERE l.doc_type = $1 AND l.doc_id = $2
ORDER BY l.at, l.id;

-- name: DocTypesInUse :many
SELECT t::text FROM unnest(@types::text[]) AS t
WHERE EXISTS (SELECT 1 FROM record.documents d WHERE d.doc_type = t);

-- name: LockRequest :exec
SELECT pg_advisory_xact_lock(hashtextextended(@request_id::text, 0));

-- name: DocumentByRequest :one
SELECT * FROM record.documents
WHERE request_id = CAST(@request_id::text AS uuid) AND requested_by IS NOT DISTINCT FROM sqlc.narg(requested_by)::bigint;

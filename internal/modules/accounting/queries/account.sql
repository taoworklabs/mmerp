-- TODO(task5): has_lines and the rule reference are false until entry_lines and rules exist.

-- name: ListAccounts :many
SELECT a.id, a.code, a.name, a.type, a.parent_id, a.active,
  NOT EXISTS (SELECT 1 FROM accounting.accounts c WHERE c.parent_id = a.id) AS postable,
  false AS has_lines
FROM accounting.accounts a ORDER BY a.code;

-- name: LockAccount :one
SELECT code, type, false AS has_lines, false AS has_rule,
  EXISTS (SELECT 1 FROM accounting.accounts c WHERE c.parent_id = a.id) AS has_children
FROM accounting.accounts a WHERE a.id = $1 FOR UPDATE;

-- name: AccountParent :one
-- The prospective parent of account @id (0 on create): its type, and whether it is @id or below it.
WITH RECURSIVE sub AS (
  SELECT x.id FROM accounting.accounts x WHERE x.id = @id::bigint
  UNION ALL
  SELECT a.id FROM accounting.accounts a JOIN sub ON a.parent_id = sub.id
)
SELECT p.type, false AS has_lines, EXISTS (SELECT 1 FROM sub WHERE sub.id = p.id) AS cyclic
FROM accounting.accounts p WHERE p.id = @parent_id::bigint;

-- name: CreateAccount :one
INSERT INTO accounting.accounts (code, name, type, parent_id, active)
VALUES (@code, @name, @type, sqlc.narg(parent_id), @active) RETURNING id;

-- name: UpdateAccount :exec
UPDATE accounting.accounts SET code = @code, name = @name, type = @type, parent_id = sqlc.narg(parent_id), active = @active
WHERE id = @id;

-- name: DeleteAccount :exec
DELETE FROM accounting.accounts WHERE id = $1;

-- name: PostableActive :many
SELECT a.id FROM accounting.accounts a
WHERE a.id = ANY(@ids::bigint[]) AND a.active
  AND NOT EXISTS (SELECT 1 FROM accounting.accounts c WHERE c.parent_id = a.id);

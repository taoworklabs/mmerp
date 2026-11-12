-- name: ListAccounts :many
SELECT a.id, a.code, a.name, a.type, a.parent_id, a.active, a.by_partner,
  NOT EXISTS (SELECT 1 FROM accounting.accounts c WHERE c.parent_id = a.id) AS postable,
  EXISTS (SELECT 1 FROM accounting.entry_lines l WHERE l.account_id = a.id) AS has_lines
FROM accounting.accounts a ORDER BY a.code;

-- name: LockAccount :one
-- tree_has_lines: lines on the account or any account below it.
WITH RECURSIVE sub AS (
  SELECT $1::bigint AS id
  UNION
  SELECT c.id FROM accounting.accounts c JOIN sub ON c.parent_id = sub.id
)
SELECT code, type, parent_id, by_partner, EXISTS (SELECT 1 FROM accounting.entry_lines l WHERE l.account_id = a.id) AS has_lines, EXISTS (SELECT 1 FROM accounting.rules r WHERE r.account_id = a.id) AS has_rule,
  EXISTS (SELECT 1 FROM accounting.accounts c WHERE c.parent_id = a.id) AS has_children,
  EXISTS (SELECT 1 FROM accounting.entry_lines l JOIN sub ON l.account_id = sub.id) AS tree_has_lines
FROM accounting.accounts a WHERE a.id = $1 FOR UPDATE OF a;

-- name: AccountParent :one
-- The prospective parent of account @id (0 on create): its type, and whether it is @id or below it.
WITH RECURSIVE sub AS (
  SELECT x.id FROM accounting.accounts x WHERE x.id = @id::bigint
  UNION
  SELECT a.id FROM accounting.accounts a JOIN sub ON a.parent_id = sub.id
)
SELECT p.type, p.by_partner, EXISTS (SELECT 1 FROM accounting.entry_lines l WHERE l.account_id = p.id) AS has_lines, EXISTS (SELECT 1 FROM sub WHERE sub.id = p.id) AS cyclic
FROM accounting.accounts p WHERE p.id = @parent_id::bigint;

-- name: CreateAccount :one
INSERT INTO accounting.accounts (code, name, type, parent_id, active, by_partner)
VALUES (@code, @name, @type, sqlc.narg(parent_id), @active, @by_partner) RETURNING id;

-- name: UpdateAccount :exec
UPDATE accounting.accounts SET code = @code, name = @name, type = @type, parent_id = sqlc.narg(parent_id), active = @active, by_partner = @by_partner
WHERE id = @id;

-- name: DeleteAccount :exec
DELETE FROM accounting.accounts WHERE id = $1;

-- name: PostableActive :many
SELECT a.id FROM accounting.accounts a
WHERE a.id = ANY(@ids::bigint[]) AND a.active
  AND NOT EXISTS (SELECT 1 FROM accounting.accounts c WHERE c.parent_id = a.id);

-- name: LockChart :exec
-- Serializes chart writes: parent checks and rule checks read rows other writes change. The two-int
-- form is a key space apart from the bigint keys of request ids.
SELECT pg_advisory_xact_lock(hashtext('accounting.chart'), 0);

-- name: LockAccounts :exec
-- An edited account and its new parent, in id order: entry writes share-lock the accounts they use.
-- Reversals do not: they touch only accounts with lines, which has_lines already guards.
SELECT 1 FROM accounting.accounts WHERE id = ANY(@ids::bigint[]) ORDER BY id FOR UPDATE;

-- name: AccountFlags :many
SELECT id, by_partner FROM accounting.accounts WHERE id = ANY(@ids::bigint[]);

-- name: RuleKindsOfAccount :many
SELECT kind FROM accounting.rules WHERE account_id = $1 ORDER BY kind;

-- name: ListRules :many
SELECT kind, account_id, side FROM accounting.rules ORDER BY kind;

-- name: SaveRule :exec
INSERT INTO accounting.rules (kind, account_id, side) VALUES (@kind, @account_id, @side)
ON CONFLICT (kind) DO UPDATE SET account_id = EXCLUDED.account_id, side = EXCLUDED.side;

-- name: DeleteRule :exec
DELETE FROM accounting.rules WHERE kind = $1;

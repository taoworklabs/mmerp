-- name: ListRules :many
-- Every kind that has a rule or a posting line; the declared kinds are added by the service.
SELECT k.kind, r.account_id, r.side
FROM (SELECT kind FROM accounting.rules UNION SELECT kind FROM posting.lines) k
LEFT JOIN accounting.rules r ON r.kind = k.kind
ORDER BY k.kind;

-- name: SaveRule :exec
INSERT INTO accounting.rules (kind, account_id, side) VALUES (@kind, @account_id, @side)
ON CONFLICT (kind) DO UPDATE SET account_id = EXCLUDED.account_id, side = EXCLUDED.side;

-- name: DeleteRule :exec
DELETE FROM accounting.rules WHERE kind = $1;

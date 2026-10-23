-- Planted violation: a module writes only the schema it owns, dependency or not.
-- name: ReserveItem :exec
UPDATE inventory.items SET reserved = reserved + $2 WHERE id = $1;

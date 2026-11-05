-- name: AnyAccount :one
SELECT EXISTS (SELECT 1 FROM accounting.accounts)::bool AS any;

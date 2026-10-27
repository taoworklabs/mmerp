-- name: GetLoginPolicy :one
SELECT min_length, upper, lower, digit, symbol, max_failures, lock_minutes FROM iam.login_policy;

-- name: SetLoginPolicy :exec
UPDATE iam.login_policy SET min_length = $1, upper = $2, lower = $3, digit = $4, symbol = $5, max_failures = $6, lock_minutes = $7;

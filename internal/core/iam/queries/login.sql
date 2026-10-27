-- name: ReserveLoginAttempt :one
-- Counts the attempt before the password is checked, in one atomic upsert, so
-- parallel attempts cannot all slip past the threshold. No row means the login
-- is locked; otherwise the flag is true only for the call that sets the lock.
INSERT INTO iam.login_failures AS f (login, failures, last_failed_at)
VALUES (lower(@login), 1, now())
ON CONFLICT (login) DO UPDATE SET
    failures = CASE WHEN f.failures + 1 >= @max_failures::int THEN 0 ELSE f.failures + 1 END,
    last_failed_at = now(),
    locked_until = CASE WHEN f.failures + 1 >= @max_failures::int THEN now() + make_interval(mins => @lock_minutes::int) ELSE f.locked_until END
WHERE f.locked_until IS NULL OR f.locked_until <= now()
RETURNING (f.failures = 0)::boolean AS locked;

-- name: DeleteStaleLoginFailures :exec
DELETE FROM iam.login_failures WHERE last_failed_at < now() - interval '1 day' AND (locked_until IS NULL OR locked_until <= now());

-- name: ClearLoginFailures :exec
DELETE FROM iam.login_failures WHERE login = lower(@login);

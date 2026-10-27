-- name: IsLoginLocked :one
SELECT EXISTS (SELECT 1 FROM iam.login_failures WHERE login = lower(@login) AND locked_until > now());

-- name: RecordLoginFailure :one
-- One atomic upsert, so concurrent attempts cannot both miss the threshold.
-- Returns true only for the call that sets the lock; no row while a lock is active.
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

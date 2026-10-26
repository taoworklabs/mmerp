-- name: DelegatorsOf :many
-- Which of @users have an active delegation to @delegate on @today; one hop only.
SELECT user_id FROM approval.delegations
WHERE delegate_id = @delegate AND user_id = ANY(@users::bigint[]) AND starts_on <= @today AND ends_on >= @today;

-- name: DelegatesOf :many
-- Who holds an active delegation from one of @users on @today.
SELECT DISTINCT delegate_id FROM approval.delegations
WHERE user_id = ANY(@users::bigint[]) AND starts_on <= @today AND ends_on >= @today;

-- name: LockDelegations :exec
-- Serialises creates for one user, so two overlapping rows can't both pass the check.
SELECT pg_advisory_xact_lock(hashtextextended('approval.delegation', @user_id::bigint));

-- name: UsersExist :one
SELECT count(*) = 2 FROM iam.users WHERE id IN (@user_id::bigint, @delegate_id::bigint);

-- name: OverlappingDelegation :one
SELECT id FROM approval.delegations
WHERE user_id = @user_id AND starts_on <= @ends_on AND ends_on >= @starts_on ORDER BY id LIMIT 1;

-- name: InsertDelegation :one
INSERT INTO approval.delegations (user_id, delegate_id, starts_on, ends_on, created_by)
VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: ListDelegations :many
-- Not yet ended; everyone's when @everyone, else @actor's own.
SELECT d.id, d.user_id, u.name AS user_name, d.delegate_id, g.name AS delegate_name, d.starts_on, d.ends_on
FROM approval.delegations d JOIN iam.users u ON u.id = d.user_id JOIN iam.users g ON g.id = d.delegate_id
WHERE d.ends_on >= @today AND (@everyone::bool OR d.user_id = @actor)
ORDER BY d.starts_on, d.id;

-- name: GetDelegation :one
SELECT * FROM approval.delegations WHERE id = $1;

-- name: DeleteDelegation :exec
DELETE FROM approval.delegations WHERE id = $1;

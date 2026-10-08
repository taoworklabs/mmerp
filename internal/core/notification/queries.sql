-- name: Insert :many
INSERT INTO notification.notifications (user_id, kind, doc_type, doc_id, job_id, actor_id)
SELECT unnest(@users::bigint[]), @kind, sqlc.narg(doc_type), sqlc.narg(doc_id), sqlc.narg(job_id), sqlc.narg(actor_id)
RETURNING id;

-- name: Mailable :many
-- The notifications to email: their user has an address and the tenant a mail server.
SELECT n.id FROM notification.notifications n JOIN iam.users u ON u.id = n.user_id
WHERE n.id = ANY(@ids::bigint[]) AND u.email IS NOT NULL AND EXISTS (SELECT 1 FROM notification.mail_server);

-- name: GetNotification :one
SELECT user_id, kind FROM notification.notifications WHERE id = $1;

-- name: List :many
SELECT n.id, n.kind, n.doc_type, n.doc_id, n.job_id, n.created_at, n.read_at, u.name AS actor_name, d.number
FROM notification.notifications n
LEFT JOIN iam.users u ON u.id = n.actor_id
LEFT JOIN record.documents d ON d.id = n.doc_id AND d.doc_type = n.doc_type
WHERE n.user_id = @user_id AND (sqlc.narg(before)::bigint IS NULL OR n.id < sqlc.narg(before))
ORDER BY n.id DESC
LIMIT @lim;

-- name: Unread :one
SELECT count(*) FROM notification.notifications WHERE user_id = $1 AND read_at IS NULL;

-- name: MarkRead :execrows
UPDATE notification.notifications SET read_at = coalesce(read_at, now()) WHERE id = $1 AND user_id = $2;

-- name: MarkAllRead :exec
UPDATE notification.notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL;

-- name: GetMailServer :one
SELECT host, port, security, username, password, from_address, base_url FROM notification.mail_server;

-- name: SaveMailServer :exec
INSERT INTO notification.mail_server (host, port, security, username, password, from_address, base_url)
VALUES (@host, @port, @security, @username, @password, @from_address, @base_url)
ON CONFLICT (id) DO UPDATE SET host = excluded.host, port = excluded.port, security = excluded.security,
    username = excluded.username, password = coalesce(excluded.password, notification.mail_server.password),
    from_address = excluded.from_address, base_url = excluded.base_url, updated_at = now();

-- name: DeleteMailServer :execrows
DELETE FROM notification.mail_server;

-- name: Recipient :one
SELECT email, locale FROM iam.users WHERE id = $1;

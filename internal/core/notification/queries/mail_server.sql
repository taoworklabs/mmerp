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

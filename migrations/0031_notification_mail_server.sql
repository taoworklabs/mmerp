-- The tenant's SMTP server, one row or none: without it no email is sent. The password is
-- sealed by the app's keyring; base_url is the address users reach mmerp at, for links.
CREATE TABLE notification.mail_server (
    id           boolean PRIMARY KEY DEFAULT true CHECK (id),
    host         text NOT NULL,
    port         int NOT NULL CHECK (port BETWEEN 1 AND 65535),
    security     text NOT NULL CHECK (security IN ('starttls', 'tls', 'none')),
    username     text NOT NULL,
    password     bytea,
    from_address text NOT NULL,
    base_url     text NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

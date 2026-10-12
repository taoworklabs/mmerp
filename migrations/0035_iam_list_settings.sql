-- A user's own settings for one list: named views (the list's query string) and the columns
-- shown with their widths. Personal display preferences, so neither audited nor shared.
CREATE TABLE iam.list_settings (
    user_id  bigint NOT NULL REFERENCES iam.users ON DELETE CASCADE,
    list_key text NOT NULL CHECK (list_key ~ '^[a-z][a-z0-9_.-]{0,99}$'),
    settings jsonb NOT NULL,
    PRIMARY KEY (user_id, list_key)
);

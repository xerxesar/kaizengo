-- Read model: permissions.user_role

CREATE TABLE IF NOT EXISTS user_roles_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    user_id    TEXT NOT NULL DEFAULT '',
    role       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_user_roles_read_user ON user_roles_read(user_id, org_id);

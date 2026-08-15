-- Read model: identity.user

CREATE TABLE IF NOT EXISTS users_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    email      TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_users_read_org ON users_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_read_email ON users_read(email);

-- Read model: permissions.role

CREATE TABLE IF NOT EXISTS roles_read (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    deleted     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    label       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_roles_read_org_name ON roles_read(org_id, name);

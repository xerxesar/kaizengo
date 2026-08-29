-- Read model: permissions.acl_entry

CREATE TABLE IF NOT EXISTS acl_entrys_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    role_id    TEXT NOT NULL DEFAULT '',
    effect     TEXT NOT NULL DEFAULT 'allow',
    resource   TEXT NOT NULL DEFAULT '',
    actions    TEXT NOT NULL DEFAULT '["*"]',
    fields     TEXT NOT NULL DEFAULT '"*"',
    domain     TEXT NOT NULL DEFAULT '[]',
    priority   INTEGER NOT NULL DEFAULT 0,
    active     BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_acl_entrys_read_org_role ON acl_entrys_read(org_id, role_id);
CREATE INDEX IF NOT EXISTS idx_acl_entrys_read_resource ON acl_entrys_read(org_id, resource);

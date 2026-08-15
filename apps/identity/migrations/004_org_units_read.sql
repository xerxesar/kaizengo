-- Read model: identity.org_unit

CREATE TABLE IF NOT EXISTS org_units_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    type       TEXT NOT NULL DEFAULT '',
    parent_id  TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_org_units_read_org ON org_units_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_org_units_read_parent ON org_units_read(org_id, parent_id);

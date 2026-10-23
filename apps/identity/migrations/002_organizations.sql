-- identity.organization

CREATE TABLE IF NOT EXISTS organizations (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    slug       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_organizations_org ON organizations(org_id, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug) WHERE deleted = false;

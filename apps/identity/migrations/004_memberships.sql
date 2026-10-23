-- identity.membership

CREATE TABLE IF NOT EXISTS memberships (
    id              TEXT PRIMARY KEY,
    org_id          TEXT NOT NULL,
    author_id       TEXT NOT NULL,
    deleted         BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    user_id         TEXT NOT NULL DEFAULT '',
    org_unit_id     TEXT NOT NULL DEFAULT '',
    role            TEXT NOT NULL DEFAULT '',
    effective_from  TEXT NOT NULL DEFAULT '',
    effective_to    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_memberships_org ON memberships(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_memberships_user ON memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_unit ON memberships(org_unit_id);

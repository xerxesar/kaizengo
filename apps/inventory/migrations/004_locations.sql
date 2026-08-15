-- Read model: hierarchical stock locations

CREATE TABLE IF NOT EXISTS locations_read (
    id             TEXT PRIMARY KEY,
    org_id         TEXT NOT NULL,
    author_id      TEXT NOT NULL,
    deleted        BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL,
    name           TEXT NOT NULL DEFAULT '',
    code           TEXT NOT NULL DEFAULT '',
    location_type  TEXT NOT NULL DEFAULT 'bin',
    parent_id      TEXT NOT NULL DEFAULT '',
    usage          TEXT NOT NULL DEFAULT 'internal',
    barcode        TEXT NOT NULL DEFAULT '',
    active         BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX IF NOT EXISTS idx_locations_read_org ON locations_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_locations_read_parent ON locations_read(org_id, parent_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_locations_read_code ON locations_read(org_id, code) WHERE deleted = false;

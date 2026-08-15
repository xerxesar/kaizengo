-- Read models: reorder rules, IoT devices, integration endpoints

CREATE TABLE IF NOT EXISTS reorder_rules_read (
    id               TEXT PRIMARY KEY,
    org_id           TEXT NOT NULL,
    author_id        TEXT NOT NULL,
    deleted          BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL,
    variant_id       TEXT NOT NULL DEFAULT '',
    location_id      TEXT NOT NULL DEFAULT '',
    method           TEXT NOT NULL DEFAULT 'safety_stock',
    min_qty          DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_qty          DOUBLE PRECISION NOT NULL DEFAULT 0,
    safety_stock     DOUBLE PRECISION NOT NULL DEFAULT 0,
    lead_time_days   INTEGER NOT NULL DEFAULT 0,
    avg_daily_demand DOUBLE PRECISION NOT NULL DEFAULT 0,
    reorder_point    DOUBLE PRECISION NOT NULL DEFAULT 0,
    qty_to_order     DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_reorder_rules_read_org ON reorder_rules_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_reorder_rules_read_variant ON reorder_rules_read(org_id, variant_id);

CREATE TABLE IF NOT EXISTS iot_devices_read (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    deleted     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    device_type TEXT NOT NULL DEFAULT 'barcode_scanner',
    identifier  TEXT NOT NULL DEFAULT '',
    location_id TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'offline',
    config      TEXT NOT NULL DEFAULT '',
    last_seen   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_iot_devices_read_org ON iot_devices_read(org_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS integrations_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    kind       TEXT NOT NULL DEFAULT 'webhook',
    endpoint   TEXT NOT NULL DEFAULT '',
    events     TEXT NOT NULL DEFAULT '',
    secret     TEXT NOT NULL DEFAULT '',
    active     BOOLEAN NOT NULL DEFAULT true,
    config     TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_integrations_read_org ON integrations_read(org_id, updated_at DESC);

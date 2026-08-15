-- Read models: on-hand, lots, serials, double-entry ledger, cost layers

CREATE TABLE IF NOT EXISTS stock_lots_read (
    id                 TEXT PRIMARY KEY,
    org_id             TEXT NOT NULL,
    author_id          TEXT NOT NULL,
    deleted            BOOLEAN NOT NULL DEFAULT false,
    created_at         TIMESTAMPTZ NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL,
    variant_id         TEXT NOT NULL DEFAULT '',
    name               TEXT NOT NULL DEFAULT '',
    expiry_date        TEXT NOT NULL DEFAULT '',
    manufactured_date  TEXT NOT NULL DEFAULT '',
    quantity           DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_stock_lots_read_org ON stock_lots_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_stock_lots_read_variant ON stock_lots_read(org_id, variant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_lots_read_name ON stock_lots_read(org_id, variant_id, name) WHERE deleted = false;

CREATE TABLE IF NOT EXISTS stock_serials_read (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    deleted     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    variant_id  TEXT NOT NULL DEFAULT '',
    lot_id      TEXT NOT NULL DEFAULT '',
    serial      TEXT NOT NULL DEFAULT '',
    location_id TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'available'
);
CREATE INDEX IF NOT EXISTS idx_stock_serials_read_org ON stock_serials_read(org_id, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_serials_read_serial ON stock_serials_read(org_id, serial) WHERE deleted = false;

CREATE TABLE IF NOT EXISTS stock_quants_read (
    id           TEXT PRIMARY KEY,
    org_id       TEXT NOT NULL,
    author_id    TEXT NOT NULL,
    deleted      BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,
    variant_id   TEXT NOT NULL DEFAULT '',
    location_id  TEXT NOT NULL DEFAULT '',
    lot_id       TEXT NOT NULL DEFAULT '',
    quantity     DOUBLE PRECISION NOT NULL DEFAULT 0,
    reserved_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
    uom_id       TEXT NOT NULL DEFAULT '',
    unit_cost    DOUBLE PRECISION NOT NULL DEFAULT 0,
    value        DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_stock_quants_read_org ON stock_quants_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_stock_quants_read_lookup ON stock_quants_read(org_id, variant_id, location_id, lot_id) WHERE deleted = false;

CREATE TABLE IF NOT EXISTS stock_ledgers_read (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    deleted     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    picking_id  TEXT NOT NULL DEFAULT '',
    move_id     TEXT NOT NULL DEFAULT '',
    variant_id  TEXT NOT NULL DEFAULT '',
    location_id TEXT NOT NULL DEFAULT '',
    lot_id      TEXT NOT NULL DEFAULT '',
    serial_id   TEXT NOT NULL DEFAULT '',
    side        TEXT NOT NULL DEFAULT 'debit',
    quantity    DOUBLE PRECISION NOT NULL DEFAULT 0,
    uom_id      TEXT NOT NULL DEFAULT '',
    unit_cost   DOUBLE PRECISION NOT NULL DEFAULT 0,
    value       DOUBLE PRECISION NOT NULL DEFAULT 0,
    occurred_at TEXT NOT NULL DEFAULT '',
    memo        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_stock_ledgers_read_org ON stock_ledgers_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_stock_ledgers_read_picking ON stock_ledgers_read(org_id, picking_id);
CREATE INDEX IF NOT EXISTS idx_stock_ledgers_read_variant ON stock_ledgers_read(org_id, variant_id, location_id);

CREATE TABLE IF NOT EXISTS cost_layers_read (
    id             TEXT PRIMARY KEY,
    org_id         TEXT NOT NULL,
    author_id      TEXT NOT NULL,
    deleted        BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL,
    variant_id     TEXT NOT NULL DEFAULT '',
    lot_id         TEXT NOT NULL DEFAULT '',
    quantity       DOUBLE PRECISION NOT NULL DEFAULT 0,
    original_qty   DOUBLE PRECISION NOT NULL DEFAULT 0,
    unit_cost      DOUBLE PRECISION NOT NULL DEFAULT 0,
    received_at    TEXT NOT NULL DEFAULT '',
    source_move_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_cost_layers_read_org ON cost_layers_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_cost_layers_read_variant ON cost_layers_read(org_id, variant_id);

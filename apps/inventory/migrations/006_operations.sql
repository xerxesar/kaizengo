-- Read models: pickings, moves, adjustments, RMAs

CREATE TABLE IF NOT EXISTS pickings_read (
    id                  TEXT PRIMARY KEY,
    org_id              TEXT NOT NULL,
    author_id           TEXT NOT NULL,
    deleted             BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL,
    name                TEXT NOT NULL DEFAULT '',
    picking_type        TEXT NOT NULL DEFAULT 'internal',
    state               TEXT NOT NULL DEFAULT 'draft',
    source_location_id  TEXT NOT NULL DEFAULT '',
    dest_location_id    TEXT NOT NULL DEFAULT '',
    scheduled_date      TEXT NOT NULL DEFAULT '',
    done_date           TEXT NOT NULL DEFAULT '',
    origin              TEXT NOT NULL DEFAULT '',
    partner_name        TEXT NOT NULL DEFAULT '',
    qc_status           TEXT NOT NULL DEFAULT 'skipped',
    wave                TEXT NOT NULL DEFAULT '',
    batch               TEXT NOT NULL DEFAULT '',
    notes               TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_pickings_read_org ON pickings_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_pickings_read_type ON pickings_read(org_id, picking_type, state);

CREATE TABLE IF NOT EXISTS stock_moves_read (
    id               TEXT PRIMARY KEY,
    org_id           TEXT NOT NULL,
    author_id        TEXT NOT NULL,
    deleted          BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL,
    picking_id       TEXT NOT NULL DEFAULT '',
    variant_id       TEXT NOT NULL DEFAULT '',
    from_location_id TEXT NOT NULL DEFAULT '',
    to_location_id   TEXT NOT NULL DEFAULT '',
    quantity         DOUBLE PRECISION NOT NULL DEFAULT 0,
    done_qty         DOUBLE PRECISION NOT NULL DEFAULT 0,
    uom_id           TEXT NOT NULL DEFAULT '',
    lot_id           TEXT NOT NULL DEFAULT '',
    serial_id        TEXT NOT NULL DEFAULT '',
    unit_cost        DOUBLE PRECISION NOT NULL DEFAULT 0,
    state            TEXT NOT NULL DEFAULT 'draft'
);
CREATE INDEX IF NOT EXISTS idx_stock_moves_read_org ON stock_moves_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_stock_moves_read_picking ON stock_moves_read(org_id, picking_id);

CREATE TABLE IF NOT EXISTS adjustments_read (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    deleted     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    location_id TEXT NOT NULL DEFAULT '',
    variant_id  TEXT NOT NULL DEFAULT '',
    lot_id      TEXT NOT NULL DEFAULT '',
    reason      TEXT NOT NULL DEFAULT 'cycle_count',
    counted_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
    system_qty  DOUBLE PRECISION NOT NULL DEFAULT 0,
    state       TEXT NOT NULL DEFAULT 'draft',
    notes       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_adjustments_read_org ON adjustments_read(org_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS rmas_read (
    id            TEXT PRIMARY KEY,
    org_id        TEXT NOT NULL,
    author_id     TEXT NOT NULL,
    deleted       BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    name          TEXT NOT NULL DEFAULT '',
    customer_name TEXT NOT NULL DEFAULT '',
    origin        TEXT NOT NULL DEFAULT '',
    variant_id    TEXT NOT NULL DEFAULT '',
    quantity      DOUBLE PRECISION NOT NULL DEFAULT 0,
    location_id   TEXT NOT NULL DEFAULT '',
    state         TEXT NOT NULL DEFAULT 'draft',
    disposition   TEXT NOT NULL DEFAULT 'restock',
    notes         TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_rmas_read_org ON rmas_read(org_id, updated_at DESC);

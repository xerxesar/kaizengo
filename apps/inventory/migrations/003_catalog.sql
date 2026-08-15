-- Read models: catalog (UoM, attributes, products, variants)

CREATE TABLE IF NOT EXISTS uom_groups_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_uom_groups_read_org ON uom_groups_read(org_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS uoms_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    symbol     TEXT NOT NULL DEFAULT '',
    group_id   TEXT NOT NULL DEFAULT '',
    ratio      DOUBLE PRECISION NOT NULL DEFAULT 1,
    rounding   DOUBLE PRECISION NOT NULL DEFAULT 0.000001,
    uom_type   TEXT NOT NULL DEFAULT 'reference'
);
CREATE INDEX IF NOT EXISTS idx_uoms_read_org ON uoms_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_uoms_read_group ON uoms_read(org_id, group_id);

CREATE TABLE IF NOT EXISTS attributes_read (
    id              TEXT PRIMARY KEY,
    org_id          TEXT NOT NULL,
    author_id       TEXT NOT NULL,
    deleted         BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    name            TEXT NOT NULL DEFAULT '',
    code            TEXT NOT NULL DEFAULT '',
    attribute_type  TEXT NOT NULL DEFAULT 'select'
);
CREATE INDEX IF NOT EXISTS idx_attributes_read_org ON attributes_read(org_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS attribute_values_read (
    id           TEXT PRIMARY KEY,
    org_id       TEXT NOT NULL,
    author_id    TEXT NOT NULL,
    deleted      BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,
    attribute_id TEXT NOT NULL DEFAULT '',
    name         TEXT NOT NULL DEFAULT '',
    code         TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_attribute_values_read_org ON attribute_values_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_attribute_values_read_attr ON attribute_values_read(org_id, attribute_id);

CREATE TABLE IF NOT EXISTS products_read (
    id                 TEXT PRIMARY KEY,
    org_id             TEXT NOT NULL,
    author_id          TEXT NOT NULL,
    deleted            BOOLEAN NOT NULL DEFAULT false,
    created_at         TIMESTAMPTZ NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL,
    name               TEXT NOT NULL DEFAULT '',
    code               TEXT NOT NULL DEFAULT '',
    description        TEXT NOT NULL DEFAULT '',
    category           TEXT NOT NULL DEFAULT '',
    tracking           TEXT NOT NULL DEFAULT 'none',
    costing_method     TEXT NOT NULL DEFAULT 'moving_average',
    dispatch_strategy  TEXT NOT NULL DEFAULT 'fifo',
    shelf_life_days    INTEGER NOT NULL DEFAULT 0,
    attribute_schema   TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'active',
    notes              TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_products_read_org ON products_read(org_id, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_read_code ON products_read(org_id, code) WHERE deleted = false;

CREATE TABLE IF NOT EXISTS product_variants_read (
    id            TEXT PRIMARY KEY,
    org_id        TEXT NOT NULL,
    author_id     TEXT NOT NULL,
    deleted       BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    product_id    TEXT NOT NULL DEFAULT '',
    sku           TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL DEFAULT '',
    upc           TEXT NOT NULL DEFAULT '',
    ean           TEXT NOT NULL DEFAULT '',
    barcode       TEXT NOT NULL DEFAULT '',
    attributes    TEXT NOT NULL DEFAULT '',
    buy_uom_id    TEXT NOT NULL DEFAULT '',
    stock_uom_id  TEXT NOT NULL DEFAULT '',
    sell_uom_id   TEXT NOT NULL DEFAULT '',
    standard_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    list_price    DOUBLE PRECISION NOT NULL DEFAULT 0,
    weight        DOUBLE PRECISION NOT NULL DEFAULT 0,
    average_cost  DOUBLE PRECISION NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'active'
);
CREATE INDEX IF NOT EXISTS idx_product_variants_read_org ON product_variants_read(org_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_product_variants_read_product ON product_variants_read(org_id, product_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_variants_read_sku ON product_variants_read(org_id, sku) WHERE deleted = false;

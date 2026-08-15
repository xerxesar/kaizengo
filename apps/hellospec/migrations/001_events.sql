-- Event store tables (per-app schema)

CREATE TABLE IF NOT EXISTS streams (
    stream_id   TEXT PRIMARY KEY,
    stream_type TEXT NOT NULL,
    version     BIGINT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS events (
    id          BIGSERIAL PRIMARY KEY,
    event_id    TEXT NOT NULL UNIQUE,
    stream_id   TEXT NOT NULL,
    stream_type TEXT NOT NULL,
    version     BIGINT NOT NULL,
    event_type  TEXT NOT NULL,
    payload     JSONB NOT NULL,
    metadata    JSONB,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(stream_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_stream_id ON events(stream_id, version);
CREATE INDEX IF NOT EXISTS idx_events_id ON events(id);

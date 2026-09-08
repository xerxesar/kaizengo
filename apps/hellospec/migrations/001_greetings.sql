-- hellospec.greeting

CREATE TABLE IF NOT EXISTS greetings (
    id            TEXT PRIMARY KEY,
    org_id        TEXT NOT NULL,
    author_id     TEXT NOT NULL,
    deleted       BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    message       TEXT NOT NULL DEFAULT '',
    mood          TEXT NOT NULL DEFAULT 'happy',
    internal_note TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_greetings_org ON greetings(org_id, updated_at DESC);

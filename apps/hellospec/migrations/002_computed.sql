-- Stored computes, topic/tag relations, and the indexes declared on greeting.

CREATE TABLE IF NOT EXISTS topics (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_topics_org ON topics (org_id, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_topics_name ON topics (org_id, name) WHERE deleted = false;

CREATE TABLE IF NOT EXISTS tags (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    name       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_tags_org ON tags (org_id, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_tags_name ON tags (org_id, name) WHERE deleted = false;

ALTER TABLE greetings ADD COLUMN IF NOT EXISTS topic_id TEXT NULL;
ALTER TABLE greetings ADD COLUMN IF NOT EXISTS length BIGINT NOT NULL DEFAULT 0;
ALTER TABLE greetings ADD COLUMN IF NOT EXISTS label TEXT NOT NULL DEFAULT '';
ALTER TABLE greetings ADD COLUMN IF NOT EXISTS topic_name TEXT NOT NULL DEFAULT '';
ALTER TABLE greetings ADD COLUMN IF NOT EXISTS shout TEXT NOT NULL DEFAULT '';

UPDATE greetings
SET length = char_length(message),
    label = message || ' · ' || mood,
    shout = upper(message)
WHERE length = 0 AND message <> '';

DO $$
BEGIN
  ALTER TABLE greetings
    ADD CONSTRAINT fk_greetings_topic_id
    FOREIGN KEY (topic_id) REFERENCES topics (id) ON DELETE SET NULL;
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_greetings_topic_id_org ON greetings (org_id, topic_id);
CREATE INDEX IF NOT EXISTS greeting_mood_topic ON greetings (mood, topic_id);
CREATE INDEX IF NOT EXISTS greeting_message_gin ON greetings USING gin (to_tsvector('simple', message));
CREATE INDEX IF NOT EXISTS greeting_formal ON greetings (mood) INCLUDE (label) WHERE mood = 'formal';
CREATE INDEX IF NOT EXISTS greeting_length ON greetings (length DESC);

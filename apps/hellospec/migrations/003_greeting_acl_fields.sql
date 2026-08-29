-- greeting.mood + greeting.internalNote (ACL demo fields)

ALTER TABLE greetings_read ADD COLUMN IF NOT EXISTS mood TEXT NOT NULL DEFAULT 'happy';
ALTER TABLE greetings_read ADD COLUMN IF NOT EXISTS internal_note TEXT NOT NULL DEFAULT '';

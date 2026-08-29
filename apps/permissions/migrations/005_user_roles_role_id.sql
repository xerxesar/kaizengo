-- user_role: switch from enum role slug to role_id FK

ALTER TABLE user_roles_read ADD COLUMN IF NOT EXISTS role_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_user_roles_read_role ON user_roles_read(role_id, org_id);

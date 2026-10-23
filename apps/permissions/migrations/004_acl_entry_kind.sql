-- permissions.acl_entry: store resource kind (query/command/menu/view/nav/model/…)

ALTER TABLE acl_entries
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_acl_entries_kind ON acl_entries(org_id, kind);

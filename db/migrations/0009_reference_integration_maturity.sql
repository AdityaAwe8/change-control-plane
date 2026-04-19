ALTER TABLE integrations
    ADD COLUMN IF NOT EXISTS instance_key TEXT NOT NULL DEFAULT 'default',
    ADD COLUMN IF NOT EXISTS scope_type TEXT NOT NULL DEFAULT 'organization',
    ADD COLUMN IF NOT EXISTS scope_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS auth_strategy TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS onboarding_status TEXT NOT NULL DEFAULT 'not_started';

UPDATE integrations
SET auth_strategy = CASE
    WHEN kind = 'github' THEN 'personal_access_token'
    ELSE ''
END
WHERE auth_strategy = '';

UPDATE integrations
SET scope_name = CASE
    WHEN scope_name <> '' THEN scope_name
    WHEN name <> '' THEN name
    ELSE kind
END
WHERE scope_name = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_integrations_kind_instance_scope
    ON integrations (organization_id, kind, instance_key);

CREATE INDEX IF NOT EXISTS idx_integrations_kind_scope_name
    ON integrations (organization_id, kind, scope_type, scope_name);

ALTER TABLE repositories
    ADD COLUMN IF NOT EXISTS source_integration_id TEXT REFERENCES integrations(id) ON DELETE SET NULL;

UPDATE repositories
SET source_integration_id = NULLIF(metadata->>'source_integration_id', '')
WHERE source_integration_id IS NULL
  AND metadata ? 'source_integration_id';

CREATE INDEX IF NOT EXISTS idx_repositories_source_integration
    ON repositories (organization_id, source_integration_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS config_sets (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    entries JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_config_sets_scope ON config_sets (organization_id, project_id, environment_id, service_id);
CREATE INDEX IF NOT EXISTS idx_config_sets_status ON config_sets (status);

ALTER TABLE releases ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id) ON DELETE SET NULL;
ALTER TABLE releases ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE releases ADD COLUMN IF NOT EXISTS summary TEXT NOT NULL DEFAULT '';
ALTER TABLE releases ADD COLUMN IF NOT EXISTS config_set_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE releases
SET name = version
WHERE name = '';

ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS release_id TEXT REFERENCES releases(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_releases_scope ON releases (organization_id, project_id, environment_id, status);
CREATE INDEX IF NOT EXISTS idx_rollout_executions_release ON rollout_executions (release_id);

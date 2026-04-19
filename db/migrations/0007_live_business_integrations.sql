ALTER TABLE integrations
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS control_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS connection_health TEXT NOT NULL DEFAULT 'unconfigured',
    ADD COLUMN IF NOT EXISTS last_tested_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';

ALTER TABLE repositories
    ALTER COLUMN project_id DROP NOT NULL;

ALTER TABLE repositories
    ADD COLUMN IF NOT EXISTS service_id TEXT REFERENCES services(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'discovered',
    ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_repositories_org_service ON repositories (organization_id, service_id);
CREATE INDEX IF NOT EXISTS idx_repositories_org_environment ON repositories (organization_id, environment_id);

CREATE TABLE IF NOT EXISTS integration_sync_runs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    integration_id TEXT NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    operation TEXT NOT NULL,
    status TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    details JSONB NOT NULL DEFAULT '[]'::jsonb,
    resource_count INTEGER NOT NULL DEFAULT 0,
    external_event_id TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_integration_sync_runs_scope
    ON integration_sync_runs (organization_id, integration_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_integration_sync_runs_external_event
    ON integration_sync_runs (integration_id, operation, external_event_id);

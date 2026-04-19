ALTER TABLE integrations
    ADD COLUMN IF NOT EXISTS schedule_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS schedule_interval_seconds INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sync_stale_after_seconds INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS next_scheduled_sync_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_sync_attempted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_sync_succeeded_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_sync_failed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS sync_claimed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS sync_consecutive_failures INTEGER NOT NULL DEFAULT 0;

UPDATE integrations
SET sync_stale_after_seconds = CASE
    WHEN kind = 'kubernetes' THEN 300
    WHEN kind = 'prometheus' THEN 300
    WHEN kind = 'github' THEN 900
    ELSE 600
END
WHERE sync_stale_after_seconds = 0;

CREATE INDEX IF NOT EXISTS idx_integrations_schedule_due
    ON integrations (organization_id, enabled, schedule_enabled, next_scheduled_sync_at);

ALTER TABLE integration_sync_runs
    ADD COLUMN IF NOT EXISTS trigger TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS error_class TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS scheduled_for TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_integration_sync_runs_trigger
    ON integration_sync_runs (integration_id, trigger, started_at DESC);

CREATE TABLE IF NOT EXISTS discovered_resources (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    integration_id TEXT NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    service_id TEXT REFERENCES services(id) ON DELETE SET NULL,
    environment_id TEXT REFERENCES environments(id) ON DELETE SET NULL,
    repository_id TEXT REFERENCES repositories(id) ON DELETE SET NULL,
    resource_type TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    external_id TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'discovered',
    health TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_discovered_resources_unique_scope
    ON discovered_resources (organization_id, integration_id, resource_type, external_id);

CREATE INDEX IF NOT EXISTS idx_discovered_resources_org_type
    ON discovered_resources (organization_id, resource_type, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_discovered_resources_service
    ON discovered_resources (organization_id, service_id, environment_id);

CREATE INDEX IF NOT EXISTS idx_status_events_org_created_desc
    ON status_events (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_status_events_rollout_created_desc
    ON status_events (organization_id, rollout_execution_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_status_events_service_created_desc
    ON status_events (organization_id, service_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_status_events_search_fts
    ON status_events
    USING GIN (to_tsvector('simple', coalesce(summary, '') || ' ' || coalesce(explanation::text, '')));

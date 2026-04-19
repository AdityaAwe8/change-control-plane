CREATE TABLE IF NOT EXISTS rollback_policies (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    environment_id TEXT REFERENCES environments(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 0,
    max_error_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    minimum_throughput DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_unhealthy_instances INTEGER NOT NULL DEFAULT -1,
    max_restart_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_verification_failures INTEGER NOT NULL DEFAULT 0,
    rollback_on_provider_failure BOOLEAN NOT NULL DEFAULT TRUE,
    rollback_on_critical_signals BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rollback_policies_scope
    ON rollback_policies (organization_id, project_id, service_id, environment_id, enabled, priority DESC, created_at DESC);

CREATE TABLE IF NOT EXISTS status_events (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    team_id TEXT REFERENCES teams(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    environment_id TEXT REFERENCES environments(id) ON DELETE CASCADE,
    rollout_execution_id TEXT REFERENCES rollout_executions(id) ON DELETE CASCADE,
    change_set_id TEXT REFERENCES change_sets(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'operations',
    severity TEXT NOT NULL DEFAULT 'info',
    previous_state TEXT NOT NULL DEFAULT '',
    new_state TEXT NOT NULL DEFAULT '',
    outcome TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL DEFAULT '',
    actor_type TEXT NOT NULL DEFAULT '',
    actor TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    automated BOOLEAN NOT NULL DEFAULT FALSE,
    summary TEXT NOT NULL DEFAULT '',
    explanation JSONB NOT NULL DEFAULT '[]'::jsonb,
    correlation_id TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_status_events_scope_created
    ON status_events (organization_id, project_id, service_id, environment_id, rollout_execution_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_status_events_resource
    ON status_events (organization_id, resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_status_events_type
    ON status_events (organization_id, event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_status_events_actor
    ON status_events (organization_id, actor_type, actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_status_events_outcome
    ON status_events (organization_id, outcome, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_status_events_search
    ON status_events
    USING GIN (to_tsvector('simple', coalesce(summary, '') || ' ' || coalesce(explanation::text, '')));

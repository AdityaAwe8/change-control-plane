ALTER TABLE projects ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE teams ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE services ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE environments ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE integrations ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMPTZ;

ALTER TABLE service_accounts ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE service_accounts ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'viewer';
ALTER TABLE service_accounts ADD COLUMN IF NOT EXISTS created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE service_accounts ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ;

ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_tokens_token_prefix ON api_tokens (token_prefix);
CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_org_url ON repositories (organization_id, url);

CREATE TABLE IF NOT EXISTS graph_relationships (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    source_integration_id TEXT REFERENCES integrations(id) ON DELETE CASCADE,
    relationship_type TEXT NOT NULL,
    from_resource_type TEXT NOT NULL,
    from_resource_id TEXT NOT NULL,
    to_resource_type TEXT NOT NULL,
    to_resource_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    last_observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, source_integration_id, relationship_type, from_resource_type, from_resource_id, to_resource_type, to_resource_id)
);

CREATE INDEX IF NOT EXISTS idx_graph_relationships_org_type ON graph_relationships (organization_id, relationship_type);
CREATE INDEX IF NOT EXISTS idx_graph_relationships_from_resource ON graph_relationships (from_resource_type, from_resource_id);
CREATE INDEX IF NOT EXISTS idx_graph_relationships_to_resource ON graph_relationships (to_resource_type, to_resource_id);

CREATE TABLE IF NOT EXISTS rollout_executions (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    rollout_plan_id TEXT NOT NULL REFERENCES rollout_plans(id) ON DELETE CASCADE,
    change_set_id TEXT NOT NULL REFERENCES change_sets(id) ON DELETE CASCADE,
    service_id TEXT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    current_step TEXT NOT NULL DEFAULT '',
    last_decision TEXT NOT NULL DEFAULT '',
    last_decision_reason TEXT NOT NULL DEFAULT '',
    last_verification_result TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rollout_executions_scope ON rollout_executions (organization_id, project_id, service_id, environment_id);
CREATE INDEX IF NOT EXISTS idx_rollout_executions_status ON rollout_executions (status);

CREATE TABLE IF NOT EXISTS verification_results (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    rollout_execution_id TEXT NOT NULL REFERENCES rollout_executions(id) ON DELETE CASCADE,
    rollout_plan_id TEXT NOT NULL REFERENCES rollout_plans(id) ON DELETE CASCADE,
    change_set_id TEXT NOT NULL REFERENCES change_sets(id) ON DELETE CASCADE,
    service_id TEXT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    outcome TEXT NOT NULL,
    decision TEXT NOT NULL,
    signals JSONB NOT NULL DEFAULT '[]'::jsonb,
    technical_signal_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    business_signal_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    summary TEXT NOT NULL DEFAULT '',
    explanation JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_verification_results_execution ON verification_results (rollout_execution_id, created_at);

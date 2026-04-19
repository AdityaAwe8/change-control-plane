ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS backend_type TEXT NOT NULL DEFAULT 'simulated';
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS backend_integration_id TEXT REFERENCES integrations(id) ON DELETE SET NULL;
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS signal_provider_type TEXT NOT NULL DEFAULT 'simulated';
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS signal_integration_id TEXT REFERENCES integrations(id) ON DELETE SET NULL;
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS backend_execution_id TEXT NOT NULL DEFAULT '';
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS backend_status TEXT NOT NULL DEFAULT '';
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS progress_percent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ;
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS last_reconciled_at TIMESTAMPTZ;
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS last_backend_sync_at TIMESTAMPTZ;
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS last_signal_sync_at TIMESTAMPTZ;
ALTER TABLE rollout_executions ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';

ALTER TABLE verification_results ADD COLUMN IF NOT EXISTS automated BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE verification_results ADD COLUMN IF NOT EXISTS decision_source TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE verification_results ADD COLUMN IF NOT EXISTS signal_snapshot_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE TABLE IF NOT EXISTS signal_snapshots (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    rollout_execution_id TEXT NOT NULL REFERENCES rollout_executions(id) ON DELETE CASCADE,
    rollout_plan_id TEXT NOT NULL REFERENCES rollout_plans(id) ON DELETE CASCADE,
    change_set_id TEXT NOT NULL REFERENCES change_sets(id) ON DELETE CASCADE,
    service_id TEXT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    provider_type TEXT NOT NULL,
    source_integration_id TEXT REFERENCES integrations(id) ON DELETE SET NULL,
    health TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    signals JSONB NOT NULL DEFAULT '[]'::jsonb,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rollout_executions_backend_scope ON rollout_executions (organization_id, backend_type, status);
CREATE INDEX IF NOT EXISTS idx_signal_snapshots_execution ON signal_snapshots (rollout_execution_id, created_at);
CREATE INDEX IF NOT EXISTS idx_signal_snapshots_provider ON signal_snapshots (organization_id, provider_type, created_at);

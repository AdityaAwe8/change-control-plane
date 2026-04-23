CREATE TABLE IF NOT EXISTS database_changes (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    change_set_id TEXT NOT NULL REFERENCES change_sets(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    datastore TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    execution_intent TEXT NOT NULL,
    compatibility TEXT NOT NULL,
    reversibility TEXT NOT NULL,
    risk_level TEXT NOT NULL,
    lock_risk BOOLEAN NOT NULL DEFAULT FALSE,
    manual_approval_required BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'defined',
    summary TEXT NOT NULL DEFAULT '',
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_database_changes_scope ON database_changes (organization_id, project_id, environment_id, service_id);
CREATE INDEX IF NOT EXISTS idx_database_changes_change_set ON database_changes (change_set_id);
CREATE INDEX IF NOT EXISTS idx_database_changes_status ON database_changes (status);

CREATE TABLE IF NOT EXISTS database_validation_checks (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    change_set_id TEXT NOT NULL REFERENCES change_sets(id) ON DELETE CASCADE,
    database_change_id TEXT REFERENCES database_changes(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    phase TEXT NOT NULL,
    check_type TEXT NOT NULL,
    read_only BOOLEAN NOT NULL DEFAULT TRUE,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    execution_mode TEXT NOT NULL DEFAULT 'manual_attestation',
    specification TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'defined',
    summary TEXT NOT NULL DEFAULT '',
    last_run_at TIMESTAMPTZ,
    last_result_summary TEXT NOT NULL DEFAULT '',
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_database_validation_checks_scope ON database_validation_checks (organization_id, project_id, environment_id, service_id);
CREATE INDEX IF NOT EXISTS idx_database_validation_checks_change_set ON database_validation_checks (change_set_id);
CREATE INDEX IF NOT EXISTS idx_database_validation_checks_database_change ON database_validation_checks (database_change_id);
CREATE INDEX IF NOT EXISTS idx_database_validation_checks_phase_status ON database_validation_checks (phase, status);

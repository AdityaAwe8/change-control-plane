CREATE TABLE IF NOT EXISTS database_connection_references (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    datastore TEXT NOT NULL,
    driver TEXT NOT NULL,
    dsn_env TEXT NOT NULL,
    read_only_capable BOOLEAN NOT NULL DEFAULT TRUE,
    status TEXT NOT NULL DEFAULT 'defined',
    summary TEXT NOT NULL DEFAULT '',
    last_tested_at TIMESTAMPTZ,
    last_error_summary TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_database_connection_refs_scope ON database_connection_references (organization_id, project_id, environment_id, service_id);
CREATE INDEX IF NOT EXISTS idx_database_connection_refs_status ON database_connection_references (status);

ALTER TABLE database_validation_checks
    ADD COLUMN IF NOT EXISTS connection_ref_id TEXT REFERENCES database_connection_references(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_database_validation_checks_connection_ref ON database_validation_checks (connection_ref_id);

CREATE TABLE IF NOT EXISTS database_validation_executions (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    change_set_id TEXT NOT NULL REFERENCES change_sets(id) ON DELETE CASCADE,
    database_change_id TEXT REFERENCES database_changes(id) ON DELETE SET NULL,
    validation_check_id TEXT NOT NULL REFERENCES database_validation_checks(id) ON DELETE CASCADE,
    connection_ref_id TEXT NOT NULL REFERENCES database_connection_references(id) ON DELETE CASCADE,
    trigger TEXT NOT NULL DEFAULT 'manual',
    execution_mode TEXT NOT NULL,
    status TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    result_details JSONB NOT NULL DEFAULT '[]'::jsonb,
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    error_class TEXT NOT NULL DEFAULT '',
    actor_type TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_database_validation_exec_scope ON database_validation_executions (organization_id, project_id, environment_id, service_id);
CREATE INDEX IF NOT EXISTS idx_database_validation_exec_check ON database_validation_executions (validation_check_id);
CREATE INDEX IF NOT EXISTS idx_database_validation_exec_connection_ref ON database_validation_executions (connection_ref_id);
CREATE INDEX IF NOT EXISTS idx_database_validation_exec_status ON database_validation_executions (status);

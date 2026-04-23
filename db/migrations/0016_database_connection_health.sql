ALTER TABLE database_connection_references
    ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'env_dsn',
    ADD COLUMN IF NOT EXISTS secret_ref TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_healthy_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_error_class TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_database_connection_refs_source_type ON database_connection_references (source_type);

CREATE TABLE IF NOT EXISTS database_connection_tests (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    connection_ref_id TEXT NOT NULL REFERENCES database_connection_references(id) ON DELETE CASCADE,
    trigger TEXT NOT NULL DEFAULT 'manual',
    status TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    details JSONB NOT NULL DEFAULT '[]'::jsonb,
    error_class TEXT NOT NULL DEFAULT '',
    actor_type TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_database_connection_tests_scope ON database_connection_tests (organization_id, project_id, environment_id, service_id);
CREATE INDEX IF NOT EXISTS idx_database_connection_tests_connection_ref ON database_connection_tests (connection_ref_id);
CREATE INDEX IF NOT EXISTS idx_database_connection_tests_status ON database_connection_tests (status);

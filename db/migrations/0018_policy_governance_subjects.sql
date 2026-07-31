ALTER TABLE policy_decisions
    ADD COLUMN IF NOT EXISTS release_id TEXT REFERENCES releases(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS config_set_id TEXT REFERENCES config_sets(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS database_change_id TEXT REFERENCES database_changes(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_policy_decisions_release_subject
    ON policy_decisions (
        organization_id,
        project_id,
        release_id,
        config_set_id,
        database_change_id,
        applies_to,
        created_at DESC
    );

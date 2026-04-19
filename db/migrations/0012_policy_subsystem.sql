ALTER TABLE policies
    ADD COLUMN IF NOT EXISTS project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS applies_to TEXT NOT NULL DEFAULT 'risk_assessment',
    ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS conditions JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_policies_scope_runtime
    ON policies (organization_id, project_id, service_id, environment_id, applies_to, enabled, priority DESC, created_at DESC);

ALTER TABLE policy_decisions
    ALTER COLUMN project_id DROP NOT NULL,
    ALTER COLUMN change_set_id DROP NOT NULL;

ALTER TABLE policy_decisions
    ADD COLUMN IF NOT EXISTS service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS policy_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS policy_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS policy_scope TEXT NOT NULL DEFAULT 'organization',
    ADD COLUMN IF NOT EXISTS applies_to TEXT NOT NULL DEFAULT 'risk_assessment',
    ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'advisory',
    ADD COLUMN IF NOT EXISTS risk_assessment_id TEXT REFERENCES risk_assessments(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS rollout_plan_id TEXT REFERENCES rollout_plans(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS rollout_execution_id TEXT REFERENCES rollout_executions(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_policy_decisions_org_created
    ON policy_decisions (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_policy_decisions_scope_lookup
    ON policy_decisions (
        organization_id,
        policy_id,
        change_set_id,
        risk_assessment_id,
        rollout_plan_id,
        rollout_execution_id,
        applies_to,
        created_at DESC
    );

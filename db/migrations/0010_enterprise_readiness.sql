CREATE TABLE IF NOT EXISTS identity_providers (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    issuer_url TEXT NOT NULL DEFAULT '',
    authorization_endpoint TEXT NOT NULL DEFAULT '',
    token_endpoint TEXT NOT NULL DEFAULT '',
    userinfo_endpoint TEXT NOT NULL DEFAULT '',
    jwks_uri TEXT NOT NULL DEFAULT '',
    client_id TEXT NOT NULL DEFAULT '',
    client_secret_env TEXT NOT NULL DEFAULT '',
    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    claim_mappings JSONB NOT NULL DEFAULT '{}'::jsonb,
    role_mappings JSONB NOT NULL DEFAULT '{}'::jsonb,
    allowed_domains JSONB NOT NULL DEFAULT '[]'::jsonb,
    default_role TEXT NOT NULL DEFAULT 'member',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'not_started',
    connection_health TEXT NOT NULL DEFAULT 'unconfigured',
    last_tested_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    last_authenticated_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, kind, name)
);

CREATE INDEX IF NOT EXISTS idx_identity_providers_org_enabled
    ON identity_providers (organization_id, enabled, updated_at DESC);

CREATE TABLE IF NOT EXISTS identity_links (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider_id TEXT NOT NULL REFERENCES identity_providers(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_subject TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider_id, external_subject),
    UNIQUE (provider_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_identity_links_user
    ON identity_links (organization_id, user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS webhook_registrations (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    integration_id TEXT NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    provider_kind TEXT NOT NULL,
    scope_identifier TEXT NOT NULL DEFAULT '',
    callback_url TEXT NOT NULL,
    external_hook_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'not_registered',
    delivery_health TEXT NOT NULL DEFAULT 'unknown',
    auto_managed BOOLEAN NOT NULL DEFAULT TRUE,
    last_registered_at TIMESTAMPTZ,
    last_validated_at TIMESTAMPTZ,
    last_delivery_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    failure_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (integration_id, provider_kind)
);

CREATE INDEX IF NOT EXISTS idx_webhook_registrations_org_status
    ON webhook_registrations (organization_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS outbox_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    claimed_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_pending
    ON outbox_events (status, next_attempt_at, created_at);

CREATE INDEX IF NOT EXISTS idx_outbox_events_org_event
    ON outbox_events (organization_id, event_type, created_at DESC);

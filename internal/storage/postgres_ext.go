package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *PostgresStore) UpdateOrganization(ctx context.Context, org types.Organization) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE organizations
		SET name = $2, tier = $3, mode = $4, metadata = $5, updated_at = $6
		WHERE id = $1
	`, org.ID, org.Name, org.Tier, org.Mode, jsonValue(org.Metadata), org.UpdatedAt)
	return err
}

func (s *PostgresStore) UpdateProject(ctx context.Context, project types.Project) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE projects
		SET name = $2, slug = $3, description = $4, adoption_mode = $5, status = $6, metadata = $7, updated_at = $8
		WHERE id = $1
	`, project.ID, project.Name, project.Slug, project.Description, project.AdoptionMode, project.Status, jsonValue(project.Metadata), project.UpdatedAt)
	return err
}

func (s *PostgresStore) UpdateTeam(ctx context.Context, team types.Team) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE teams
		SET name = $2, slug = $3, owner_user_ids = $4, status = $5, metadata = $6, updated_at = $7
		WHERE id = $1
	`, team.ID, team.Name, team.Slug, jsonValue(team.OwnerUserIDs), team.Status, jsonValue(team.Metadata), team.UpdatedAt)
	return err
}

func (s *PostgresStore) UpdateService(ctx context.Context, service types.Service) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE services
		SET team_id = $2, name = $3, slug = $4, description = $5, criticality = $6, tier = $7,
			customer_facing = $8, has_slo = $9, has_observability = $10, regulated_zone = $11,
			dependent_services_count = $12, status = $13, metadata = $14, updated_at = $15
		WHERE id = $1
	`, service.ID, service.TeamID, service.Name, service.Slug, service.Description, service.Criticality, service.Tier,
		service.CustomerFacing, service.HasSLO, service.HasObservability, service.RegulatedZone,
		service.DependentServicesCount, service.Status, jsonValue(service.Metadata), service.UpdatedAt)
	return err
}

func (s *PostgresStore) UpdateEnvironment(ctx context.Context, environment types.Environment) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE environments
		SET name = $2, slug = $3, type = $4, region = $5, production = $6, compliance_zone = $7,
			status = $8, metadata = $9, updated_at = $10
		WHERE id = $1
	`, environment.ID, environment.Name, environment.Slug, environment.Type, environment.Region, environment.Production, environment.ComplianceZone,
		environment.Status, jsonValue(environment.Metadata), environment.UpdatedAt)
	return err
}

func (s *PostgresStore) GetRiskAssessment(ctx context.Context, id string) (types.RiskAssessment, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, change_set_id, service_id, environment_id, score, level, confidence_score,
			explanation, blast_radius, recommended_approval_level, recommended_rollout_strategy,
			recommended_deployment_window, recommended_guardrails, metadata, created_at, updated_at
		FROM risk_assessments
		WHERE id = $1
	`, id)
	return scanRiskAssessment(row)
}

func (s *PostgresStore) GetRolloutPlan(ctx context.Context, id string) (types.RolloutPlan, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, change_set_id, risk_assessment_id, strategy, approval_required,
			approval_level, deployment_window, additional_verification, rollback_precheck_required,
			business_hours_restriction, off_hours_preferred, verification_signals, rollback_conditions,
			guardrails, steps, explanation, metadata, created_at, updated_at
		FROM rollout_plans
		WHERE id = $1
	`, id)
	return scanRolloutPlan(row)
}

func (s *PostgresStore) GetIntegration(ctx context.Context, id string) (types.Integration, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, name, kind, instance_key, scope_type, scope_name, mode, auth_strategy, onboarding_status, status, enabled, control_enabled, connection_health,
			capabilities, description, last_tested_at, last_synced_at, last_error,
			schedule_enabled, schedule_interval_seconds, sync_stale_after_seconds, next_scheduled_sync_at,
			last_sync_attempted_at, last_sync_succeeded_at, last_sync_failed_at, sync_claimed_at, sync_consecutive_failures,
			metadata, created_at, updated_at
		FROM integrations
		WHERE id = $1
	`, id)
	return scanIntegration(row)
}

func (s *PostgresStore) UpdateIntegration(ctx context.Context, integration types.Integration) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE integrations
		SET name = $2, instance_key = $3, scope_type = $4, scope_name = $5, mode = $6, auth_strategy = $7, onboarding_status = $8,
			status = $9, enabled = $10, control_enabled = $11, connection_health = $12, capabilities = $13, description = $14,
			last_tested_at = $15, last_synced_at = $16, last_error = $17, schedule_enabled = $18, schedule_interval_seconds = $19,
			sync_stale_after_seconds = $20, next_scheduled_sync_at = $21, last_sync_attempted_at = $22, last_sync_succeeded_at = $23,
			last_sync_failed_at = $24, sync_claimed_at = $25, sync_consecutive_failures = $26, metadata = $27, updated_at = $28
		WHERE id = $1
	`, integration.ID, integration.Name, integration.InstanceKey, integration.ScopeType, integration.ScopeName, integration.Mode, integration.AuthStrategy, integration.OnboardingStatus, integration.Status, integration.Enabled, integration.ControlEnabled, integration.ConnectionHealth, jsonValue(integration.Capabilities), integration.Description, integration.LastTestedAt, integration.LastSyncedAt, integration.LastError, integration.ScheduleEnabled, integration.ScheduleIntervalSeconds, integration.SyncStaleAfterSeconds, integration.NextScheduledSyncAt, integration.LastSyncAttemptedAt, integration.LastSyncSucceededAt, integration.LastSyncFailedAt, integration.SyncClaimedAt, integration.SyncConsecutiveFailures, jsonValue(integration.Metadata), integration.UpdatedAt)
	return err
}

func (s *PostgresStore) ClaimIntegrationSync(ctx context.Context, id string, dueBefore, staleClaimBefore, claimedAt time.Time) (bool, error) {
	result, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE integrations
		SET sync_claimed_at = $2, updated_at = GREATEST(updated_at, $2)
		WHERE id = $1
			AND enabled = TRUE
			AND schedule_enabled = TRUE
			AND next_scheduled_sync_at IS NOT NULL
			AND next_scheduled_sync_at <= $3
			AND (sync_claimed_at IS NULL OR sync_claimed_at < $4)
	`, id, claimedAt, dueBefore, staleClaimBefore)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

func (s *PostgresStore) CreateIntegrationSyncRun(ctx context.Context, run types.IntegrationSyncRun) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO integration_sync_runs (
			id, organization_id, integration_id, operation, trigger, status, summary, details, resource_count,
			external_event_id, error_class, scheduled_for, metadata, started_at, completed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, run.ID, run.OrganizationID, run.IntegrationID, run.Operation, run.Trigger, run.Status, run.Summary, jsonValue(run.Details), run.ResourceCount, run.ExternalEventID, run.ErrorClass, run.ScheduledFor, jsonValue(run.Metadata), run.StartedAt, run.CompletedAt, run.CreatedAt, run.UpdatedAt)
	return err
}

func (s *PostgresStore) ListIntegrationSyncRuns(ctx context.Context, query IntegrationSyncRunQuery) ([]types.IntegrationSyncRun, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, integration_id, operation, trigger, status, summary, details, resource_count,
			external_event_id, error_class, scheduled_for, metadata, started_at, completed_at, created_at, updated_at
		FROM integration_sync_runs`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("integration_id", query.IntegrationID),
		filterEqual("operation", query.Operation),
		filterEqual("trigger", query.Trigger),
		filterEqual("status", query.Status),
		filterEqual("external_event_id", query.ExternalEventID),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.IntegrationSyncRun
	for rows.Next() {
		item, err := scanIntegrationSyncRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpsertRepository(ctx context.Context, repository types.Repository) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO repositories (
			id, organization_id, project_id, service_id, environment_id, source_integration_id, name, provider, url, default_branch,
			status, last_synced_at, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			service_id = EXCLUDED.service_id,
			environment_id = EXCLUDED.environment_id,
			source_integration_id = EXCLUDED.source_integration_id,
			name = EXCLUDED.name,
			provider = EXCLUDED.provider,
			url = EXCLUDED.url,
			default_branch = EXCLUDED.default_branch,
			status = EXCLUDED.status,
			last_synced_at = EXCLUDED.last_synced_at,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, repository.ID, repository.OrganizationID, nullIfEmpty(repository.ProjectID), nullIfEmpty(repository.ServiceID), nullIfEmpty(repository.EnvironmentID), nullIfEmpty(repository.SourceIntegrationID), repository.Name, repository.Provider, repository.URL, repository.DefaultBranch, repository.Status, repository.LastSyncedAt, jsonValue(repository.Metadata), repository.CreatedAt, repository.UpdatedAt)
	return err
}

func (s *PostgresStore) GetRepository(ctx context.Context, id string) (types.Repository, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, service_id, environment_id, source_integration_id, name, provider, url, default_branch,
			status, last_synced_at, metadata, created_at, updated_at
		FROM repositories
		WHERE id = $1
	`, id)
	return scanRepository(row)
}

func (s *PostgresStore) GetRepositoryByURL(ctx context.Context, organizationID, url string) (types.Repository, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, service_id, environment_id, source_integration_id, name, provider, url, default_branch,
			status, last_synced_at, metadata, created_at, updated_at
		FROM repositories
		WHERE organization_id = $1 AND url = $2
	`, organizationID, url)
	return scanRepository(row)
}

func (s *PostgresStore) ListRepositories(ctx context.Context, query RepositoryQuery) ([]types.Repository, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, service_id, environment_id, source_integration_id, name, provider, url, default_branch,
			status, last_synced_at, metadata, created_at, updated_at
		FROM repositories`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("service_id", query.ServiceID),
		filterEqual("environment_id", query.EnvironmentID),
		filterEqual("source_integration_id", query.SourceIntegrationID),
		filterEqual("provider", query.Provider),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.Repository
	for rows.Next() {
		item, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateRepository(ctx context.Context, repository types.Repository) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE repositories
		SET project_id = $2, service_id = $3, environment_id = $4, source_integration_id = $5, name = $6, provider = $7, url = $8,
			default_branch = $9, status = $10, last_synced_at = $11, metadata = $12, updated_at = $13
		WHERE id = $1
	`, repository.ID, nullIfEmpty(repository.ProjectID), nullIfEmpty(repository.ServiceID), nullIfEmpty(repository.EnvironmentID), nullIfEmpty(repository.SourceIntegrationID), repository.Name, repository.Provider, repository.URL, repository.DefaultBranch, repository.Status, repository.LastSyncedAt, jsonValue(repository.Metadata), repository.UpdatedAt)
	return err
}

func (s *PostgresStore) UpsertDiscoveredResource(ctx context.Context, resource types.DiscoveredResource) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO discovered_resources (
			id, organization_id, integration_id, project_id, service_id, environment_id, repository_id,
			resource_type, provider, external_id, namespace, name, status, health, summary, last_seen_at,
			metadata, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19
		)
		ON CONFLICT (id) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			service_id = EXCLUDED.service_id,
			environment_id = EXCLUDED.environment_id,
			repository_id = EXCLUDED.repository_id,
			status = EXCLUDED.status,
			health = EXCLUDED.health,
			summary = EXCLUDED.summary,
			last_seen_at = EXCLUDED.last_seen_at,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, resource.ID, resource.OrganizationID, resource.IntegrationID, nullIfEmpty(resource.ProjectID), nullIfEmpty(resource.ServiceID), nullIfEmpty(resource.EnvironmentID), nullIfEmpty(resource.RepositoryID), resource.ResourceType, resource.Provider, resource.ExternalID, resource.Namespace, resource.Name, resource.Status, resource.Health, resource.Summary, resource.LastSeenAt, jsonValue(resource.Metadata), resource.CreatedAt, resource.UpdatedAt)
	return err
}

func (s *PostgresStore) GetDiscoveredResource(ctx context.Context, id string) (types.DiscoveredResource, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, integration_id, project_id, service_id, environment_id, repository_id,
			resource_type, provider, external_id, namespace, name, status, health, summary, last_seen_at,
			metadata, created_at, updated_at
		FROM discovered_resources
		WHERE id = $1
	`, id)
	return scanDiscoveredResource(row)
}

func (s *PostgresStore) ListDiscoveredResources(ctx context.Context, query DiscoveredResourceQuery) ([]types.DiscoveredResource, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, integration_id, project_id, service_id, environment_id, repository_id,
			resource_type, provider, external_id, namespace, name, status, health, summary, last_seen_at,
			metadata, created_at, updated_at
		FROM discovered_resources`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("integration_id", query.IntegrationID),
		filterEqual("resource_type", query.ResourceType),
		filterEqual("provider", query.Provider),
		filterEqual("project_id", query.ProjectID),
		filterEqual("service_id", query.ServiceID),
		filterEqual("environment_id", query.EnvironmentID),
		filterEqual("repository_id", query.RepositoryID),
		filterEqual("status", query.Status),
		filterUnmappedResources(query.UnmappedOnly),
		filterDiscoveredResourceSearch(query.Search),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.DiscoveredResource
	for rows.Next() {
		item, err := scanDiscoveredResource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateDiscoveredResource(ctx context.Context, resource types.DiscoveredResource) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE discovered_resources
		SET project_id = $2, service_id = $3, environment_id = $4, repository_id = $5, status = $6,
			health = $7, summary = $8, last_seen_at = $9, metadata = $10, updated_at = $11
		WHERE id = $1
	`, resource.ID, nullIfEmpty(resource.ProjectID), nullIfEmpty(resource.ServiceID), nullIfEmpty(resource.EnvironmentID), nullIfEmpty(resource.RepositoryID), resource.Status, resource.Health, resource.Summary, resource.LastSeenAt, jsonValue(resource.Metadata), resource.UpdatedAt)
	return err
}

func (s *PostgresStore) UpsertGraphRelationship(ctx context.Context, relationship types.GraphRelationship) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO graph_relationships (
			id, organization_id, project_id, source_integration_id, relationship_type, from_resource_type,
			from_resource_id, to_resource_type, to_resource_id, status, last_observed_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (id) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			source_integration_id = EXCLUDED.source_integration_id,
			status = EXCLUDED.status,
			last_observed_at = EXCLUDED.last_observed_at,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, relationship.ID, relationship.OrganizationID, nullIfEmpty(relationship.ProjectID), nullIfEmpty(relationship.SourceIntegrationID), relationship.RelationshipType, relationship.FromResourceType,
		relationship.FromResourceID, relationship.ToResourceType, relationship.ToResourceID, relationship.Status, relationship.LastObservedAt, jsonValue(relationship.Metadata), relationship.CreatedAt, relationship.UpdatedAt)
	return err
}

func (s *PostgresStore) ListGraphRelationships(ctx context.Context, query GraphRelationshipQuery) ([]types.GraphRelationship, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, source_integration_id, relationship_type, from_resource_type,
			from_resource_id, to_resource_type, to_resource_id, status, last_observed_at, metadata, created_at, updated_at
		FROM graph_relationships`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("source_integration_id", query.SourceIntegrationID),
		filterEqual("relationship_type", query.RelationshipType),
		filterEqual("from_resource_id", query.FromResourceID),
		filterEqual("to_resource_id", query.ToResourceID),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.GraphRelationship
	for rows.Next() {
		item, err := scanGraphRelationship(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateServiceAccount(ctx context.Context, serviceAccount types.ServiceAccount) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO service_accounts (id, organization_id, name, description, role, created_by_user_id, status, last_used_at, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, serviceAccount.ID, serviceAccount.OrganizationID, serviceAccount.Name, serviceAccount.Description, serviceAccount.Role, nullIfEmpty(serviceAccount.CreatedByUserID), serviceAccount.Status, serviceAccount.LastUsedAt, jsonValue(serviceAccount.Metadata), serviceAccount.CreatedAt, serviceAccount.UpdatedAt)
	return err
}

func (s *PostgresStore) GetServiceAccount(ctx context.Context, id string) (types.ServiceAccount, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, name, description, role, created_by_user_id, status, last_used_at, metadata, created_at, updated_at
		FROM service_accounts
		WHERE id = $1
	`, id)
	return scanServiceAccount(row)
}

func (s *PostgresStore) ListServiceAccounts(ctx context.Context, query ServiceAccountQuery) ([]types.ServiceAccount, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, name, description, role, created_by_user_id, status, last_used_at, metadata, created_at, updated_at FROM service_accounts`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("status", query.Status),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.ServiceAccount
	for rows.Next() {
		item, err := scanServiceAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateServiceAccount(ctx context.Context, serviceAccount types.ServiceAccount) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE service_accounts
		SET name = $2, description = $3, role = $4, status = $5, last_used_at = $6, metadata = $7, updated_at = $8
		WHERE id = $1
	`, serviceAccount.ID, serviceAccount.Name, serviceAccount.Description, serviceAccount.Role, serviceAccount.Status, serviceAccount.LastUsedAt, jsonValue(serviceAccount.Metadata), serviceAccount.UpdatedAt)
	return err
}

func (s *PostgresStore) CreateAPIToken(ctx context.Context, token types.APIToken) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO api_tokens (
			id, organization_id, user_id, service_account_id, name, token_prefix, token_hash, status,
			last_used_at, revoked_at, expires_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14
		)
	`, token.ID, token.OrganizationID, nullIfEmpty(token.UserID), nullIfEmpty(token.ServiceAccountID), token.Name, token.TokenPrefix, token.TokenHash, token.Status,
		token.LastUsedAt, token.RevokedAt, token.ExpiresAt, jsonValue(token.Metadata), token.CreatedAt, token.UpdatedAt)
	return err
}

func (s *PostgresStore) GetAPIToken(ctx context.Context, id string) (types.APIToken, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, user_id, service_account_id, name, token_prefix, token_hash, status, last_used_at, revoked_at, expires_at, metadata, created_at, updated_at
		FROM api_tokens
		WHERE id = $1
	`, id)
	return scanAPIToken(row)
}

func (s *PostgresStore) GetAPITokenByPrefix(ctx context.Context, prefix string) (types.APIToken, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, user_id, service_account_id, name, token_prefix, token_hash, status, last_used_at, revoked_at, expires_at, metadata, created_at, updated_at
		FROM api_tokens
		WHERE token_prefix = $1
	`, prefix)
	return scanAPIToken(row)
}

func (s *PostgresStore) ListAPITokens(ctx context.Context, query APITokenQuery) ([]types.APIToken, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, user_id, service_account_id, name, token_prefix, token_hash, status, last_used_at, revoked_at, expires_at, metadata, created_at, updated_at FROM api_tokens`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("service_account_id", query.ServiceAccountID),
		filterEqual("status", query.Status),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.APIToken
	for rows.Next() {
		item, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateAPIToken(ctx context.Context, token types.APIToken) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE api_tokens
		SET name = $2, token_hash = $3, status = $4, last_used_at = $5, revoked_at = $6, expires_at = $7, metadata = $8, updated_at = $9
		WHERE id = $1
	`, token.ID, token.Name, token.TokenHash, token.Status, token.LastUsedAt, token.RevokedAt, token.ExpiresAt, jsonValue(token.Metadata), token.UpdatedAt)
	return err
}

func (s *PostgresStore) CreateRolloutExecution(ctx context.Context, execution types.RolloutExecution) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO rollout_executions (
			id, organization_id, project_id, rollout_plan_id, change_set_id, service_id, environment_id,
			backend_type, backend_integration_id, signal_provider_type, signal_integration_id, backend_execution_id, backend_status,
			progress_percent, status, current_step, last_decision, last_decision_reason, last_verification_result,
			submitted_at, started_at, completed_at, last_reconciled_at, last_backend_sync_at, last_signal_sync_at, last_error,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25, $26,
			$27, $28, $29
		)
	`, execution.ID, execution.OrganizationID, execution.ProjectID, execution.RolloutPlanID, execution.ChangeSetID, execution.ServiceID, execution.EnvironmentID,
		execution.BackendType, nullIfEmpty(execution.BackendIntegrationID), execution.SignalProviderType, nullIfEmpty(execution.SignalIntegrationID), execution.BackendExecutionID, execution.BackendStatus,
		execution.ProgressPercent, execution.Status, execution.CurrentStep, execution.LastDecision, execution.LastDecisionReason, nullIfEmpty(execution.LastVerificationResult),
		execution.SubmittedAt, execution.StartedAt, execution.CompletedAt, execution.LastReconciledAt, execution.LastBackendSyncAt, execution.LastSignalSyncAt, execution.LastError,
		jsonValue(execution.Metadata), execution.CreatedAt, execution.UpdatedAt)
	return err
}

func (s *PostgresStore) GetRolloutExecution(ctx context.Context, id string) (types.RolloutExecution, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, rollout_plan_id, change_set_id, service_id, environment_id,
			backend_type, backend_integration_id, signal_provider_type, signal_integration_id, backend_execution_id, backend_status,
			progress_percent, status, current_step, last_decision, last_decision_reason, last_verification_result,
			submitted_at, started_at, completed_at, last_reconciled_at, last_backend_sync_at, last_signal_sync_at, last_error,
			metadata, created_at, updated_at
		FROM rollout_executions
		WHERE id = $1
	`, id)
	return scanRolloutExecution(row)
}

func (s *PostgresStore) ListRolloutExecutions(ctx context.Context, query RolloutExecutionQuery) ([]types.RolloutExecution, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, rollout_plan_id, change_set_id, service_id, environment_id,
			backend_type, backend_integration_id, signal_provider_type, signal_integration_id, backend_execution_id, backend_status,
			progress_percent, status, current_step, last_decision, last_decision_reason, last_verification_result,
			submitted_at, started_at, completed_at, last_reconciled_at, last_backend_sync_at, last_signal_sync_at, last_error,
			metadata, created_at, updated_at
		FROM rollout_executions`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("service_id", query.ServiceID),
		filterEqual("environment_id", query.EnvironmentID),
		filterEqual("status", query.Status),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.RolloutExecution
	for rows.Next() {
		item, err := scanRolloutExecution(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateRolloutExecution(ctx context.Context, execution types.RolloutExecution) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE rollout_executions
		SET backend_type = $2, backend_integration_id = $3, signal_provider_type = $4, signal_integration_id = $5, backend_execution_id = $6, backend_status = $7,
			progress_percent = $8, status = $9, current_step = $10, last_decision = $11, last_decision_reason = $12, last_verification_result = $13,
			submitted_at = $14, started_at = $15, completed_at = $16, last_reconciled_at = $17, last_backend_sync_at = $18, last_signal_sync_at = $19,
			last_error = $20, metadata = $21, updated_at = $22
		WHERE id = $1
	`, execution.ID, execution.BackendType, nullIfEmpty(execution.BackendIntegrationID), execution.SignalProviderType, nullIfEmpty(execution.SignalIntegrationID), execution.BackendExecutionID, execution.BackendStatus,
		execution.ProgressPercent, execution.Status, execution.CurrentStep, execution.LastDecision, execution.LastDecisionReason, nullIfEmpty(execution.LastVerificationResult),
		execution.SubmittedAt, execution.StartedAt, execution.CompletedAt, execution.LastReconciledAt, execution.LastBackendSyncAt, execution.LastSignalSyncAt,
		execution.LastError, jsonValue(execution.Metadata), execution.UpdatedAt)
	return err
}

func (s *PostgresStore) ClaimRolloutExecution(ctx context.Context, id string, staleBefore, claimedAt time.Time) (bool, error) {
	result, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE rollout_executions
		SET last_reconciled_at = $2, updated_at = GREATEST(updated_at, $2)
		WHERE id = $1 AND (last_reconciled_at IS NULL OR last_reconciled_at < $3)
	`, id, claimedAt, staleBefore)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

func (s *PostgresStore) CreateVerificationResult(ctx context.Context, result types.VerificationResult) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO verification_results (
			id, organization_id, project_id, rollout_execution_id, rollout_plan_id, change_set_id, service_id, environment_id,
			status, outcome, decision, signals, technical_signal_summary, business_signal_summary, automated, decision_source, signal_snapshot_ids, summary, explanation,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22
		)
	`, result.ID, result.OrganizationID, result.ProjectID, result.RolloutExecutionID, result.RolloutPlanID, result.ChangeSetID, result.ServiceID, result.EnvironmentID,
		result.Status, result.Outcome, result.Decision, jsonValue(result.Signals), jsonValue(result.TechnicalSignalSummary), jsonValue(result.BusinessSignalSummary), result.Automated, result.DecisionSource, jsonValue(result.SignalSnapshotIDs), result.Summary, jsonValue(result.Explanation),
		jsonValue(result.Metadata), result.CreatedAt, result.UpdatedAt)
	return err
}

func (s *PostgresStore) ListVerificationResults(ctx context.Context, query VerificationResultQuery) ([]types.VerificationResult, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, rollout_execution_id, rollout_plan_id, change_set_id, service_id, environment_id,
			status, outcome, decision, signals, technical_signal_summary, business_signal_summary, automated, decision_source, signal_snapshot_ids, summary, explanation,
			metadata, created_at, updated_at
		FROM verification_results`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("rollout_execution_id", query.RolloutExecutionID),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.VerificationResult
	for rows.Next() {
		item, err := scanVerificationResult(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateSignalSnapshot(ctx context.Context, snapshot types.SignalSnapshot) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO signal_snapshots (
			id, organization_id, project_id, rollout_execution_id, rollout_plan_id, change_set_id, service_id, environment_id,
			provider_type, source_integration_id, health, summary, signals, window_start, window_end,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18
		)
	`, snapshot.ID, snapshot.OrganizationID, snapshot.ProjectID, snapshot.RolloutExecutionID, snapshot.RolloutPlanID, snapshot.ChangeSetID, snapshot.ServiceID, snapshot.EnvironmentID,
		snapshot.ProviderType, nullIfEmpty(snapshot.SourceIntegrationID), snapshot.Health, snapshot.Summary, jsonValue(snapshot.Signals), snapshot.WindowStart, snapshot.WindowEnd,
		jsonValue(snapshot.Metadata), snapshot.CreatedAt, snapshot.UpdatedAt)
	return err
}

func (s *PostgresStore) ListSignalSnapshots(ctx context.Context, query SignalSnapshotQuery) ([]types.SignalSnapshot, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, rollout_execution_id, rollout_plan_id, change_set_id, service_id, environment_id,
			provider_type, source_integration_id, health, summary, signals, window_start, window_end,
			metadata, created_at, updated_at
		FROM signal_snapshots`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("rollout_execution_id", query.RolloutExecutionID),
		filterEqual("provider_type", query.ProviderType),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.SignalSnapshot
	for rows.Next() {
		item, err := scanSignalSnapshot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanRepository(row scanner) (types.Repository, error) {
	var item types.Repository
	var projectID sql.NullString
	var serviceID sql.NullString
	var environmentID sql.NullString
	var sourceIntegrationID sql.NullString
	var lastSyncedAt sql.NullTime
	var metadata []byte
	err := row.Scan(
		&item.ID,
		&item.OrganizationID,
		&projectID,
		&serviceID,
		&environmentID,
		&sourceIntegrationID,
		&item.Name,
		&item.Provider,
		&item.URL,
		&item.DefaultBranch,
		&item.Status,
		&lastSyncedAt,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.ProjectID = projectID.String
	item.ServiceID = serviceID.String
	item.EnvironmentID = environmentID.String
	item.SourceIntegrationID = sourceIntegrationID.String
	if lastSyncedAt.Valid {
		item.LastSyncedAt = &lastSyncedAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanIntegrationSyncRun(row scanner) (types.IntegrationSyncRun, error) {
	var item types.IntegrationSyncRun
	var details []byte
	var metadata []byte
	var completedAt sql.NullTime
	var scheduledFor sql.NullTime
	err := row.Scan(
		&item.ID,
		&item.OrganizationID,
		&item.IntegrationID,
		&item.Operation,
		&item.Trigger,
		&item.Status,
		&item.Summary,
		&details,
		&item.ResourceCount,
		&item.ExternalEventID,
		&item.ErrorClass,
		&scheduledFor,
		&metadata,
		&item.StartedAt,
		&completedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	if scheduledFor.Valid {
		item.ScheduledFor = &scheduledFor.Time
	}
	_ = json.Unmarshal(details, &item.Details)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanDiscoveredResource(row scanner) (types.DiscoveredResource, error) {
	var item types.DiscoveredResource
	var projectID sql.NullString
	var serviceID sql.NullString
	var environmentID sql.NullString
	var repositoryID sql.NullString
	var lastSeenAt sql.NullTime
	var metadata []byte
	err := row.Scan(
		&item.ID,
		&item.OrganizationID,
		&item.IntegrationID,
		&projectID,
		&serviceID,
		&environmentID,
		&repositoryID,
		&item.ResourceType,
		&item.Provider,
		&item.ExternalID,
		&item.Namespace,
		&item.Name,
		&item.Status,
		&item.Health,
		&item.Summary,
		&lastSeenAt,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.ProjectID = projectID.String
	item.ServiceID = serviceID.String
	item.EnvironmentID = environmentID.String
	item.RepositoryID = repositoryID.String
	if lastSeenAt.Valid {
		item.LastSeenAt = &lastSeenAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanGraphRelationship(row scanner) (types.GraphRelationship, error) {
	var item types.GraphRelationship
	var projectID sql.NullString
	var sourceIntegrationID sql.NullString
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &projectID, &sourceIntegrationID, &item.RelationshipType, &item.FromResourceType,
		&item.FromResourceID, &item.ToResourceType, &item.ToResourceID, &item.Status, &item.LastObservedAt, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.ProjectID = projectID.String
	item.SourceIntegrationID = sourceIntegrationID.String
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanServiceAccount(row scanner) (types.ServiceAccount, error) {
	var item types.ServiceAccount
	var createdByUserID sql.NullString
	var lastUsedAt sql.NullTime
	var metadata []byte
	err := row.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Description, &item.Role, &createdByUserID, &item.Status, &lastUsedAt, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.CreatedByUserID = createdByUserID.String
	if lastUsedAt.Valid {
		item.LastUsedAt = &lastUsedAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanAPIToken(row scanner) (types.APIToken, error) {
	var item types.APIToken
	var userID sql.NullString
	var serviceAccountID sql.NullString
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime
	var expiresAt sql.NullTime
	var metadata []byte
	err := row.Scan(&item.ID, &item.OrganizationID, &userID, &serviceAccountID, &item.Name, &item.TokenPrefix, &item.TokenHash, &item.Status, &lastUsedAt, &revokedAt, &expiresAt, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.UserID = userID.String
	item.ServiceAccountID = serviceAccountID.String
	if lastUsedAt.Valid {
		item.LastUsedAt = &lastUsedAt.Time
	}
	if revokedAt.Valid {
		item.RevokedAt = &revokedAt.Time
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanRolloutExecution(row scanner) (types.RolloutExecution, error) {
	var item types.RolloutExecution
	var backendIntegrationID sql.NullString
	var signalIntegrationID sql.NullString
	var submittedAt sql.NullTime
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	var lastReconciledAt sql.NullTime
	var lastBackendSyncAt sql.NullTime
	var lastSignalSyncAt sql.NullTime
	var lastVerificationResult sql.NullString
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.RolloutPlanID, &item.ChangeSetID, &item.ServiceID, &item.EnvironmentID,
		&item.BackendType, &backendIntegrationID, &item.SignalProviderType, &signalIntegrationID, &item.BackendExecutionID, &item.BackendStatus,
		&item.ProgressPercent, &item.Status, &item.CurrentStep, &item.LastDecision, &item.LastDecisionReason, &lastVerificationResult,
		&submittedAt, &startedAt, &completedAt, &lastReconciledAt, &lastBackendSyncAt, &lastSignalSyncAt, &item.LastError,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.BackendIntegrationID = backendIntegrationID.String
	item.SignalIntegrationID = signalIntegrationID.String
	item.LastVerificationResult = lastVerificationResult.String
	if submittedAt.Valid {
		item.SubmittedAt = &submittedAt.Time
	}
	if startedAt.Valid {
		item.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	if lastReconciledAt.Valid {
		item.LastReconciledAt = &lastReconciledAt.Time
	}
	if lastBackendSyncAt.Valid {
		item.LastBackendSyncAt = &lastBackendSyncAt.Time
	}
	if lastSignalSyncAt.Valid {
		item.LastSignalSyncAt = &lastSignalSyncAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanVerificationResult(row scanner) (types.VerificationResult, error) {
	var item types.VerificationResult
	var signals []byte
	var technicalSummary []byte
	var businessSummary []byte
	var signalSnapshotIDs []byte
	var explanation []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.RolloutExecutionID, &item.RolloutPlanID, &item.ChangeSetID, &item.ServiceID, &item.EnvironmentID,
		&item.Status, &item.Outcome, &item.Decision, &signals, &technicalSummary, &businessSummary, &item.Automated, &item.DecisionSource, &signalSnapshotIDs, &item.Summary, &explanation,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	_ = json.Unmarshal(signals, &item.Signals)
	_ = json.Unmarshal(technicalSummary, &item.TechnicalSignalSummary)
	_ = json.Unmarshal(businessSummary, &item.BusinessSignalSummary)
	_ = json.Unmarshal(signalSnapshotIDs, &item.SignalSnapshotIDs)
	_ = json.Unmarshal(explanation, &item.Explanation)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanSignalSnapshot(row scanner) (types.SignalSnapshot, error) {
	var item types.SignalSnapshot
	var sourceIntegrationID sql.NullString
	var signals []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.RolloutExecutionID, &item.RolloutPlanID, &item.ChangeSetID, &item.ServiceID, &item.EnvironmentID,
		&item.ProviderType, &sourceIntegrationID, &item.Health, &item.Summary, &signals, &item.WindowStart, &item.WindowEnd,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.SourceIntegrationID = sourceIntegrationID.String
	_ = json.Unmarshal(signals, &item.Signals)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

package storage

import (
	"context"
	"database/sql"
	"encoding/json"

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
		SELECT id, organization_id, name, kind, mode, status, capabilities, description, last_synced_at, metadata, created_at, updated_at
		FROM integrations
		WHERE id = $1
	`, id)
	return scanIntegration(row)
}

func (s *PostgresStore) UpdateIntegration(ctx context.Context, integration types.Integration) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE integrations
		SET name = $2, mode = $3, status = $4, capabilities = $5, description = $6, last_synced_at = $7, metadata = $8, updated_at = $9
		WHERE id = $1
	`, integration.ID, integration.Name, integration.Mode, integration.Status, jsonValue(integration.Capabilities), integration.Description, integration.LastSyncedAt, jsonValue(integration.Metadata), integration.UpdatedAt)
	return err
}

func (s *PostgresStore) UpsertRepository(ctx context.Context, repository types.Repository) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO repositories (id, organization_id, project_id, name, provider, url, default_branch, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			name = EXCLUDED.name,
			provider = EXCLUDED.provider,
			url = EXCLUDED.url,
			default_branch = EXCLUDED.default_branch,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, repository.ID, repository.OrganizationID, repository.ProjectID, repository.Name, repository.Provider, repository.URL, repository.DefaultBranch, jsonValue(repository.Metadata), repository.CreatedAt, repository.UpdatedAt)
	return err
}

func (s *PostgresStore) GetRepositoryByURL(ctx context.Context, organizationID, url string) (types.Repository, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, name, provider, url, default_branch, metadata, created_at, updated_at
		FROM repositories
		WHERE organization_id = $1 AND url = $2
	`, organizationID, url)
	return scanRepository(row)
}

func (s *PostgresStore) ListRepositories(ctx context.Context, query RepositoryQuery) ([]types.Repository, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, name, provider, url, default_branch, metadata, created_at, updated_at FROM repositories`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
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
			status, current_step, last_decision, last_decision_reason, last_verification_result,
			started_at, completed_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17
		)
	`, execution.ID, execution.OrganizationID, execution.ProjectID, execution.RolloutPlanID, execution.ChangeSetID, execution.ServiceID, execution.EnvironmentID,
		execution.Status, execution.CurrentStep, execution.LastDecision, execution.LastDecisionReason, nullIfEmpty(execution.LastVerificationResult),
		execution.StartedAt, execution.CompletedAt, jsonValue(execution.Metadata), execution.CreatedAt, execution.UpdatedAt)
	return err
}

func (s *PostgresStore) GetRolloutExecution(ctx context.Context, id string) (types.RolloutExecution, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, rollout_plan_id, change_set_id, service_id, environment_id,
			status, current_step, last_decision, last_decision_reason, last_verification_result,
			started_at, completed_at, metadata, created_at, updated_at
		FROM rollout_executions
		WHERE id = $1
	`, id)
	return scanRolloutExecution(row)
}

func (s *PostgresStore) ListRolloutExecutions(ctx context.Context, query RolloutExecutionQuery) ([]types.RolloutExecution, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, rollout_plan_id, change_set_id, service_id, environment_id,
			status, current_step, last_decision, last_decision_reason, last_verification_result,
			started_at, completed_at, metadata, created_at, updated_at
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
		SET status = $2, current_step = $3, last_decision = $4, last_decision_reason = $5, last_verification_result = $6,
			started_at = $7, completed_at = $8, metadata = $9, updated_at = $10
		WHERE id = $1
	`, execution.ID, execution.Status, execution.CurrentStep, execution.LastDecision, execution.LastDecisionReason, nullIfEmpty(execution.LastVerificationResult),
		execution.StartedAt, execution.CompletedAt, jsonValue(execution.Metadata), execution.UpdatedAt)
	return err
}

func (s *PostgresStore) CreateVerificationResult(ctx context.Context, result types.VerificationResult) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO verification_results (
			id, organization_id, project_id, rollout_execution_id, rollout_plan_id, change_set_id, service_id, environment_id,
			status, outcome, decision, signals, technical_signal_summary, business_signal_summary, summary, explanation,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19
		)
	`, result.ID, result.OrganizationID, result.ProjectID, result.RolloutExecutionID, result.RolloutPlanID, result.ChangeSetID, result.ServiceID, result.EnvironmentID,
		result.Status, result.Outcome, result.Decision, jsonValue(result.Signals), jsonValue(result.TechnicalSignalSummary), jsonValue(result.BusinessSignalSummary), result.Summary, jsonValue(result.Explanation),
		jsonValue(result.Metadata), result.CreatedAt, result.UpdatedAt)
	return err
}

func (s *PostgresStore) ListVerificationResults(ctx context.Context, query VerificationResultQuery) ([]types.VerificationResult, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, rollout_execution_id, rollout_plan_id, change_set_id, service_id, environment_id,
			status, outcome, decision, signals, technical_signal_summary, business_signal_summary, summary, explanation,
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

func scanRepository(row scanner) (types.Repository, error) {
	var item types.Repository
	var metadata []byte
	err := row.Scan(&item.ID, &item.OrganizationID, &item.ProjectID, &item.Name, &item.Provider, &item.URL, &item.DefaultBranch, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
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
		return item, err
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
		return item, err
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
		return item, err
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
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	var lastVerificationResult sql.NullString
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.RolloutPlanID, &item.ChangeSetID, &item.ServiceID, &item.EnvironmentID,
		&item.Status, &item.CurrentStep, &item.LastDecision, &item.LastDecisionReason, &lastVerificationResult,
		&startedAt, &completedAt, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	item.LastVerificationResult = lastVerificationResult.String
	if startedAt.Valid {
		item.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanVerificationResult(row scanner) (types.VerificationResult, error) {
	var item types.VerificationResult
	var signals []byte
	var technicalSummary []byte
	var businessSummary []byte
	var explanation []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.RolloutExecutionID, &item.RolloutPlanID, &item.ChangeSetID, &item.ServiceID, &item.EnvironmentID,
		&item.Status, &item.Outcome, &item.Decision, &signals, &technicalSummary, &businessSummary, &item.Summary, &explanation,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(signals, &item.Signals)
	_ = json.Unmarshal(technicalSummary, &item.TechnicalSignalSummary)
	_ = json.Unmarshal(businessSummary, &item.BusinessSignalSummary)
	_ = json.Unmarshal(explanation, &item.Explanation)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

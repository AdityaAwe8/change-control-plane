package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type txKey struct{}

type queryRunner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(cfg common.Config) (*PostgresStore, error) {
	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if cfg.AutoMigrate {
		if err := ApplyMigrations(context.Background(), db, filepath.Join("db", "migrations")); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) CreateOrganization(ctx context.Context, org types.Organization) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO organizations (id, name, slug, tier, mode, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, org.ID, org.Name, org.Slug, org.Tier, org.Mode, jsonValue(org.Metadata), org.CreatedAt, org.UpdatedAt)
	return err
}

func (s *PostgresStore) GetOrganization(ctx context.Context, id string) (types.Organization, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, name, slug, tier, mode, metadata, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`, id)
	return scanOrganization(row)
}

func (s *PostgresStore) GetOrganizationBySlug(ctx context.Context, slug string) (types.Organization, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, name, slug, tier, mode, metadata, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`, slug)
	return scanOrganization(row)
}

func (s *PostgresStore) ListOrganizations(ctx context.Context, query OrganizationQuery) ([]types.Organization, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, name, slug, tier, mode, metadata, created_at, updated_at FROM organizations`,
		query.Limit,
		query.Offset,
		filterIDs("id", query.IDs),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.Organization
	for rows.Next() {
		item, err := scanOrganization(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateProject(ctx context.Context, project types.Project) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO projects (id, organization_id, name, slug, description, adoption_mode, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, project.ID, project.OrganizationID, project.Name, project.Slug, project.Description, project.AdoptionMode, project.Status, jsonValue(project.Metadata), project.CreatedAt, project.UpdatedAt)
	return err
}

func (s *PostgresStore) GetProject(ctx context.Context, id string) (types.Project, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, name, slug, description, adoption_mode, status, metadata, created_at, updated_at
		FROM projects WHERE id = $1
	`, id)
	return scanProject(row)
}

func (s *PostgresStore) ListProjects(ctx context.Context, query ProjectQuery) ([]types.Project, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, name, slug, description, adoption_mode, status, metadata, created_at, updated_at FROM projects`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterIDs("id", query.IDs),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.Project
	for rows.Next() {
		item, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateTeam(ctx context.Context, team types.Team) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO teams (id, organization_id, project_id, name, slug, owner_user_ids, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, team.ID, team.OrganizationID, team.ProjectID, team.Name, team.Slug, jsonValue(team.OwnerUserIDs), team.Status, jsonValue(team.Metadata), team.CreatedAt, team.UpdatedAt)
	return err
}

func (s *PostgresStore) GetTeam(ctx context.Context, id string) (types.Team, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, name, slug, owner_user_ids, status, metadata, created_at, updated_at
		FROM teams WHERE id = $1
	`, id)
	return scanTeam(row)
}

func (s *PostgresStore) ListTeams(ctx context.Context, query TeamQuery) ([]types.Team, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, name, slug, owner_user_ids, status, metadata, created_at, updated_at FROM teams`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterIDs("id", query.IDs),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.Team
	for rows.Next() {
		item, err := scanTeam(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateService(ctx context.Context, service types.Service) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO services (
			id, organization_id, project_id, team_id, name, slug, description, criticality, tier,
			customer_facing, has_slo, has_observability, regulated_zone, dependent_services_count, status,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15,
			$16, $17, $18
		)
	`, service.ID, service.OrganizationID, service.ProjectID, service.TeamID, service.Name, service.Slug, service.Description, service.Criticality, service.Tier,
		service.CustomerFacing, service.HasSLO, service.HasObservability, service.RegulatedZone, service.DependentServicesCount, service.Status,
		jsonValue(service.Metadata), service.CreatedAt, service.UpdatedAt)
	return err
}

func (s *PostgresStore) GetService(ctx context.Context, id string) (types.Service, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, team_id, name, slug, description, criticality, tier,
			customer_facing, has_slo, has_observability, regulated_zone, dependent_services_count, status,
			metadata, created_at, updated_at
		FROM services WHERE id = $1
	`, id)
	return scanService(row)
}

func (s *PostgresStore) ListServices(ctx context.Context, query ServiceQuery) ([]types.Service, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, team_id, name, slug, description, criticality, tier,
			customer_facing, has_slo, has_observability, regulated_zone, dependent_services_count, status,
			metadata, created_at, updated_at
		FROM services`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("team_id", query.TeamID),
		filterIDs("id", query.IDs),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.Service
	for rows.Next() {
		item, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateEnvironment(ctx context.Context, environment types.Environment) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO environments (id, organization_id, project_id, name, slug, type, region, production, compliance_zone, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, environment.ID, environment.OrganizationID, environment.ProjectID, environment.Name, environment.Slug, environment.Type, environment.Region, environment.Production, environment.ComplianceZone, environment.Status, jsonValue(environment.Metadata), environment.CreatedAt, environment.UpdatedAt)
	return err
}

func (s *PostgresStore) GetEnvironment(ctx context.Context, id string) (types.Environment, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, name, slug, type, region, production, compliance_zone, status, metadata, created_at, updated_at
		FROM environments WHERE id = $1
	`, id)
	return scanEnvironment(row)
}

func (s *PostgresStore) ListEnvironments(ctx context.Context, query EnvironmentQuery) ([]types.Environment, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, name, slug, type, region, production, compliance_zone, status, metadata, created_at, updated_at FROM environments`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterIDs("id", query.IDs),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.Environment
	for rows.Next() {
		item, err := scanEnvironment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateChangeSet(ctx context.Context, change types.ChangeSet) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO change_sets (
			id, organization_id, project_id, service_id, environment_id, summary, change_types, file_count, resource_count,
			touches_infrastructure, touches_iam, touches_secrets, touches_schema, dependency_changes,
			historical_incident_count, poor_rollback_history, status, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20
		)
	`, change.ID, change.OrganizationID, change.ProjectID, change.ServiceID, change.EnvironmentID, change.Summary, jsonValue(change.ChangeTypes), change.FileCount, change.ResourceCount,
		change.TouchesInfrastructure, change.TouchesIAM, change.TouchesSecrets, change.TouchesSchema, change.DependencyChanges,
		change.HistoricalIncidentCount, change.PoorRollbackHistory, change.Status, jsonValue(change.Metadata), change.CreatedAt, change.UpdatedAt)
	return err
}

func (s *PostgresStore) GetChangeSet(ctx context.Context, id string) (types.ChangeSet, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, service_id, environment_id, summary, change_types, file_count, resource_count,
			touches_infrastructure, touches_iam, touches_secrets, touches_schema, dependency_changes,
			historical_incident_count, poor_rollback_history, status, metadata, created_at, updated_at
		FROM change_sets WHERE id = $1
	`, id)
	return scanChangeSet(row)
}

func (s *PostgresStore) ListChangeSets(ctx context.Context, query ChangeSetQuery) ([]types.ChangeSet, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, service_id, environment_id, summary, change_types, file_count, resource_count,
			touches_infrastructure, touches_iam, touches_secrets, touches_schema, dependency_changes,
			historical_incident_count, poor_rollback_history, status, metadata, created_at, updated_at
		FROM change_sets`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("service_id", query.ServiceID),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.ChangeSet
	for rows.Next() {
		item, err := scanChangeSet(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateRiskAssessment(ctx context.Context, assessment types.RiskAssessment) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO risk_assessments (
			id, organization_id, project_id, change_set_id, service_id, environment_id, score, level, confidence_score,
			explanation, blast_radius, recommended_approval_level, recommended_rollout_strategy,
			recommended_deployment_window, recommended_guardrails, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17, $18
		)
	`, assessment.ID, assessment.OrganizationID, assessment.ProjectID, assessment.ChangeSetID, assessment.ServiceID, assessment.EnvironmentID, assessment.Score, assessment.Level, assessment.ConfidenceScore,
		jsonValue(assessment.Explanation), jsonValue(assessment.BlastRadius), assessment.RecommendedApprovalLevel, assessment.RecommendedRolloutStrategy,
		assessment.RecommendedDeploymentWindow, jsonValue(assessment.RecommendedGuardrails), jsonValue(assessment.Metadata), assessment.CreatedAt, assessment.UpdatedAt)
	return err
}

func (s *PostgresStore) ListRiskAssessments(ctx context.Context, query RiskAssessmentQuery) ([]types.RiskAssessment, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, change_set_id, service_id, environment_id, score, level, confidence_score,
			explanation, blast_radius, recommended_approval_level, recommended_rollout_strategy,
			recommended_deployment_window, recommended_guardrails, metadata, created_at, updated_at
		FROM risk_assessments`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("change_set_id", query.ChangeSetID),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.RiskAssessment
	for rows.Next() {
		item, err := scanRiskAssessment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateRolloutPlan(ctx context.Context, plan types.RolloutPlan) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO rollout_plans (
			id, organization_id, project_id, change_set_id, risk_assessment_id, strategy, approval_required,
			approval_level, deployment_window, additional_verification, rollback_precheck_required,
			business_hours_restriction, off_hours_preferred, verification_signals, rollback_conditions,
			guardrails, steps, explanation, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21
		)
	`, plan.ID, plan.OrganizationID, plan.ProjectID, plan.ChangeSetID, plan.RiskAssessmentID, plan.Strategy, plan.ApprovalRequired,
		plan.ApprovalLevel, plan.DeploymentWindow, plan.AdditionalVerification, plan.RollbackPrecheckRequired,
		plan.BusinessHoursRestriction, plan.OffHoursPreferred, jsonValue(plan.VerificationSignals), jsonValue(plan.RollbackConditions),
		jsonValue(plan.Guardrails), jsonValue(plan.Steps), jsonValue(plan.Explanation), jsonValue(plan.Metadata), plan.CreatedAt, plan.UpdatedAt)
	return err
}

func (s *PostgresStore) ListRolloutPlans(ctx context.Context, query RolloutPlanQuery) ([]types.RolloutPlan, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, change_set_id, risk_assessment_id, strategy, approval_required,
			approval_level, deployment_window, additional_verification, rollback_precheck_required,
			business_hours_restriction, off_hours_preferred, verification_signals, rollback_conditions,
			guardrails, steps, explanation, metadata, created_at, updated_at
		FROM rollout_plans`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("change_set_id", query.ChangeSetID),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.RolloutPlan
	for rows.Next() {
		item, err := scanRolloutPlan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateAuditEvent(ctx context.Context, event types.AuditEvent) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO audit_events (
			id, organization_id, project_id, actor_id, actor_type, actor, action, resource_type, resource_id, outcome,
			details, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14
		)
	`, event.ID, nullIfEmpty(event.OrganizationID), nullIfEmpty(event.ProjectID), event.ActorID, event.ActorType, event.Actor, event.Action, event.ResourceType, event.ResourceID, event.Outcome,
		jsonValue(event.Details), jsonValue(event.Metadata), event.CreatedAt, event.UpdatedAt)
	return err
}

func (s *PostgresStore) ListAuditEvents(ctx context.Context, query AuditEventQuery) ([]types.AuditEvent, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, actor_id, actor_type, actor, action, resource_type, resource_id, outcome,
			details, metadata, created_at, updated_at
		FROM audit_events`,
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
	var items []types.AuditEvent
	for rows.Next() {
		item, err := scanAuditEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateIntegration(ctx context.Context, integration types.Integration) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO integrations (id, organization_id, name, kind, mode, status, capabilities, description, last_synced_at, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, integration.ID, nullIfEmpty(integration.OrganizationID), integration.Name, integration.Kind, integration.Mode, integration.Status, jsonValue(integration.Capabilities), integration.Description, integration.LastSyncedAt, jsonValue(integration.Metadata), integration.CreatedAt, integration.UpdatedAt)
	return err
}

func (s *PostgresStore) UpsertIntegration(ctx context.Context, integration types.Integration) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO integrations (id, organization_id, name, kind, mode, status, capabilities, description, last_synced_at, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			organization_id = EXCLUDED.organization_id,
			name = EXCLUDED.name,
			kind = EXCLUDED.kind,
			mode = EXCLUDED.mode,
			status = EXCLUDED.status,
			capabilities = EXCLUDED.capabilities,
			description = EXCLUDED.description,
			last_synced_at = EXCLUDED.last_synced_at,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, integration.ID, nullIfEmpty(integration.OrganizationID), integration.Name, integration.Kind, integration.Mode, integration.Status, jsonValue(integration.Capabilities), integration.Description, integration.LastSyncedAt, jsonValue(integration.Metadata), integration.CreatedAt, integration.UpdatedAt)
	return err
}

func (s *PostgresStore) ListIntegrations(ctx context.Context, query IntegrationQuery) ([]types.Integration, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, name, kind, mode, status, capabilities, description, last_synced_at, metadata, created_at, updated_at FROM integrations`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.Integration
	for rows.Next() {
		item, err := scanIntegration(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateUser(ctx context.Context, user types.User) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO users (id, organization_id, email, display_name, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.ID, user.OrganizationID, user.Email, user.DisplayName, user.Status, jsonValue(user.Metadata), user.CreatedAt, user.UpdatedAt)
	return err
}

func (s *PostgresStore) GetUser(ctx context.Context, id string) (types.User, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, email, display_name, status, metadata, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	return scanUser(row)
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (types.User, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, email, display_name, status, metadata, created_at, updated_at
		FROM users WHERE email = $1
	`, email)
	return scanUser(row)
}

func (s *PostgresStore) CreateOrganizationMembership(ctx context.Context, membership types.OrganizationMembership) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO organization_memberships (id, user_id, organization_id, role, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, membership.ID, membership.UserID, membership.OrganizationID, membership.Role, membership.Status, jsonValue(membership.Metadata), membership.CreatedAt, membership.UpdatedAt)
	return err
}

func (s *PostgresStore) GetOrganizationMembership(ctx context.Context, userID, organizationID string) (types.OrganizationMembership, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, user_id, organization_id, role, status, metadata, created_at, updated_at
		FROM organization_memberships
		WHERE user_id = $1 AND organization_id = $2
	`, userID, organizationID)
	return scanOrganizationMembership(row)
}

func (s *PostgresStore) ListOrganizationMembershipsByUser(ctx context.Context, userID string) ([]types.OrganizationMembership, error) {
	rows, err := s.runner(ctx).QueryContext(ctx, `
		SELECT id, user_id, organization_id, role, status, metadata, created_at, updated_at
		FROM organization_memberships
		WHERE user_id = $1
		ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.OrganizationMembership
	for rows.Next() {
		item, err := scanOrganizationMembership(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateProjectMembership(ctx context.Context, membership types.ProjectMembership) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO project_memberships (id, user_id, organization_id, project_id, role, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, membership.ID, membership.UserID, membership.OrganizationID, membership.ProjectID, membership.Role, membership.Status, jsonValue(membership.Metadata), membership.CreatedAt, membership.UpdatedAt)
	return err
}

func (s *PostgresStore) GetProjectMembership(ctx context.Context, userID, projectID string) (types.ProjectMembership, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, user_id, organization_id, project_id, role, status, metadata, created_at, updated_at
		FROM project_memberships
		WHERE user_id = $1 AND project_id = $2
	`, userID, projectID)
	return scanProjectMembership(row)
}

func (s *PostgresStore) ListProjectMembershipsByUser(ctx context.Context, userID string) ([]types.ProjectMembership, error) {
	rows, err := s.runner(ctx).QueryContext(ctx, `
		SELECT id, user_id, organization_id, project_id, role, status, metadata, created_at, updated_at
		FROM project_memberships
		WHERE user_id = $1
		ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.ProjectMembership
	for rows.Next() {
		item, err := scanProjectMembership(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) runner(ctx context.Context) queryRunner {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return s.db
}

type condition struct {
	sql    string
	values []any
}

func buildListQuery(base string, limit, offset int, filters ...condition) (string, []any) {
	query := base
	args := make([]any, 0, 8)
	clauses := make([]string, 0, len(filters))
	for _, filter := range filters {
		if filter.sql == "" {
			continue
		}
		rewritten := filter.sql
		for idx := range filter.values {
			placeholder := fmt.Sprintf("$%d", idx+1)
			rewritten = strings.Replace(rewritten, placeholder, fmt.Sprintf("$%d", len(args)+idx+1), 1)
		}
		clauses = append(clauses, rewritten)
		args = append(args, filter.values...)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at"
	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}
	return query, args
}

func filterEqual(column, value string) condition {
	if value == "" {
		return condition{}
	}
	return condition{sql: fmt.Sprintf("%s = $1", column), values: []any{value}}
}

func filterIDs(column string, ids []string) condition {
	if len(ids) == 0 {
		return condition{}
	}
	parts := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for index, id := range ids {
		parts = append(parts, fmt.Sprintf("$%d", index+1))
		args = append(args, id)
	}
	return condition{
		sql:    fmt.Sprintf("%s IN (%s)", column, strings.Join(parts, ", ")),
		values: args,
	}
}

func jsonValue(value any) []byte {
	if value == nil {
		return []byte(`{}`)
	}
	switch typed := value.(type) {
	case []string:
		payload, _ := json.Marshal(typed)
		return payload
	case []types.RolloutStep:
		payload, _ := json.Marshal(typed)
		return payload
	case types.BlastRadius:
		payload, _ := json.Marshal(typed)
		return payload
	default:
		payload, _ := json.Marshal(value)
		return payload
	}
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type scanner interface {
	Scan(dest ...any) error
}

func scanOrganization(row scanner) (types.Organization, error) {
	var item types.Organization
	var metadata []byte
	err := row.Scan(&item.ID, &item.Name, &item.Slug, &item.Tier, &item.Mode, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanProject(row scanner) (types.Project, error) {
	var item types.Project
	var metadata []byte
	err := row.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Slug, &item.Description, &item.AdoptionMode, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanTeam(row scanner) (types.Team, error) {
	var item types.Team
	var owners []byte
	var metadata []byte
	err := row.Scan(&item.ID, &item.OrganizationID, &item.ProjectID, &item.Name, &item.Slug, &owners, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(owners, &item.OwnerUserIDs)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanService(row scanner) (types.Service, error) {
	var item types.Service
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.TeamID, &item.Name, &item.Slug, &item.Description, &item.Criticality, &item.Tier,
		&item.CustomerFacing, &item.HasSLO, &item.HasObservability, &item.RegulatedZone, &item.DependentServicesCount, &item.Status,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanEnvironment(row scanner) (types.Environment, error) {
	var item types.Environment
	var metadata []byte
	err := row.Scan(&item.ID, &item.OrganizationID, &item.ProjectID, &item.Name, &item.Slug, &item.Type, &item.Region, &item.Production, &item.ComplianceZone, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanChangeSet(row scanner) (types.ChangeSet, error) {
	var item types.ChangeSet
	var changeTypes []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.ServiceID, &item.EnvironmentID, &item.Summary, &changeTypes, &item.FileCount, &item.ResourceCount,
		&item.TouchesInfrastructure, &item.TouchesIAM, &item.TouchesSecrets, &item.TouchesSchema, &item.DependencyChanges,
		&item.HistoricalIncidentCount, &item.PoorRollbackHistory, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(changeTypes, &item.ChangeTypes)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanRiskAssessment(row scanner) (types.RiskAssessment, error) {
	var item types.RiskAssessment
	var explanation []byte
	var blastRadius []byte
	var guardrails []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.ChangeSetID, &item.ServiceID, &item.EnvironmentID, &item.Score, &item.Level, &item.ConfidenceScore,
		&explanation, &blastRadius, &item.RecommendedApprovalLevel, &item.RecommendedRolloutStrategy,
		&item.RecommendedDeploymentWindow, &guardrails, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(explanation, &item.Explanation)
	_ = json.Unmarshal(blastRadius, &item.BlastRadius)
	_ = json.Unmarshal(guardrails, &item.RecommendedGuardrails)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanRolloutPlan(row scanner) (types.RolloutPlan, error) {
	var item types.RolloutPlan
	var verificationSignals []byte
	var rollbackConditions []byte
	var guardrails []byte
	var steps []byte
	var explanation []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.ProjectID, &item.ChangeSetID, &item.RiskAssessmentID, &item.Strategy, &item.ApprovalRequired,
		&item.ApprovalLevel, &item.DeploymentWindow, &item.AdditionalVerification, &item.RollbackPrecheckRequired,
		&item.BusinessHoursRestriction, &item.OffHoursPreferred, &verificationSignals, &rollbackConditions,
		&guardrails, &steps, &explanation, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(verificationSignals, &item.VerificationSignals)
	_ = json.Unmarshal(rollbackConditions, &item.RollbackConditions)
	_ = json.Unmarshal(guardrails, &item.Guardrails)
	_ = json.Unmarshal(steps, &item.Steps)
	_ = json.Unmarshal(explanation, &item.Explanation)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanAuditEvent(row scanner) (types.AuditEvent, error) {
	var item types.AuditEvent
	var organizationID sql.NullString
	var projectID sql.NullString
	var details []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &organizationID, &projectID, &item.ActorID, &item.ActorType, &item.Actor, &item.Action, &item.ResourceType, &item.ResourceID, &item.Outcome,
		&details, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	item.OrganizationID = organizationID.String
	item.ProjectID = projectID.String
	_ = json.Unmarshal(details, &item.Details)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanIntegration(row scanner) (types.Integration, error) {
	var item types.Integration
	var organizationID sql.NullString
	var lastSyncedAt sql.NullTime
	var capabilities []byte
	var metadata []byte
	err := row.Scan(&item.ID, &organizationID, &item.Name, &item.Kind, &item.Mode, &item.Status, &capabilities, &item.Description, &lastSyncedAt, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.OrganizationID = organizationID.String
	if lastSyncedAt.Valid {
		item.LastSyncedAt = &lastSyncedAt.Time
	}
	_ = json.Unmarshal(capabilities, &item.Capabilities)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanUser(row scanner) (types.User, error) {
	var item types.User
	var metadata []byte
	err := row.Scan(&item.ID, &item.OrganizationID, &item.Email, &item.DisplayName, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanOrganizationMembership(row scanner) (types.OrganizationMembership, error) {
	var item types.OrganizationMembership
	var metadata []byte
	err := row.Scan(&item.ID, &item.UserID, &item.OrganizationID, &item.Role, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanProjectMembership(row scanner) (types.ProjectMembership, error) {
	var item types.ProjectMembership
	var metadata []byte
	err := row.Scan(&item.ID, &item.UserID, &item.OrganizationID, &item.ProjectID, &item.Role, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

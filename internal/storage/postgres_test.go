package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestPostgresStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("CCP_TEST_DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("CCP_DB_DSN")
	}
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/change_control_plane?sslmode=disable"
	}

	cfg := common.LoadConfig()
	cfg.DBDSN = dsn
	cfg.AutoMigrate = true

	store, err := NewPostgresStore(cfg)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer store.Close()

	if err := truncateTables(store.db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	now := time.Now().UTC()

	org := types.Organization{
		BaseRecord: types.BaseRecord{ID: "org_test_pg", CreatedAt: now, UpdatedAt: now},
		Name:       "Acme",
		Slug:       "acme-pg",
		Tier:       "growth",
		Mode:       "startup",
	}
	if err := store.CreateOrganization(ctx, org); err != nil {
		t.Fatal(err)
	}

	user := types.User{
		BaseRecord:     types.BaseRecord{ID: "user_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		Email:          "owner@acme.local",
		DisplayName:    "Acme Owner",
		Status:         "active",
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	orgMembership := types.OrganizationMembership{
		BaseRecord:     types.BaseRecord{ID: "orgm_test_pg", CreatedAt: now, UpdatedAt: now},
		UserID:         user.ID,
		OrganizationID: org.ID,
		Role:           "org_admin",
		Status:         "active",
	}
	if err := store.CreateOrganizationMembership(ctx, orgMembership); err != nil {
		t.Fatal(err)
	}

	project := types.Project{
		BaseRecord:     types.BaseRecord{ID: "proj_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		Name:           "Platform",
		Slug:           "platform",
		Description:    "Core platform",
		AdoptionMode:   "advisory",
		Status:         "active",
	}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}

	projectMembership := types.ProjectMembership{
		BaseRecord:     types.BaseRecord{ID: "prjm_test_pg", CreatedAt: now, UpdatedAt: now},
		UserID:         user.ID,
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		Role:           "project_admin",
		Status:         "active",
	}
	if err := store.CreateProjectMembership(ctx, projectMembership); err != nil {
		t.Fatal(err)
	}

	team := types.Team{
		BaseRecord:     types.BaseRecord{ID: "team_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		Name:           "Platform Team",
		Slug:           "platform-team",
		OwnerUserIDs:   []string{user.ID},
		Status:         "active",
	}
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatal(err)
	}

	service := types.Service{
		BaseRecord:             types.BaseRecord{ID: "svc_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:         org.ID,
		ProjectID:              project.ID,
		TeamID:                 team.ID,
		Name:                   "Checkout API",
		Slug:                   "checkout-api",
		Description:            "Payments surface",
		Criticality:            "mission_critical",
		Tier:                   "service",
		CustomerFacing:         true,
		HasSLO:                 true,
		HasObservability:       true,
		DependentServicesCount: 2,
		Status:                 "active",
	}
	if err := store.CreateService(ctx, service); err != nil {
		t.Fatal(err)
	}

	environment := types.Environment{
		BaseRecord:     types.BaseRecord{ID: "env_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
		Status:         "active",
	}
	if err := store.CreateEnvironment(ctx, environment); err != nil {
		t.Fatal(err)
	}

	change := types.ChangeSet{
		BaseRecord:            types.BaseRecord{ID: "chg_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:        org.ID,
		ProjectID:             project.ID,
		ServiceID:             service.ID,
		EnvironmentID:         environment.ID,
		Summary:               "Apply checkout retry change",
		ChangeTypes:           []string{"code", "iam"},
		FileCount:             12,
		ResourceCount:         1,
		TouchesInfrastructure: true,
		Status:                "ingested",
	}
	if err := store.CreateChangeSet(ctx, change); err != nil {
		t.Fatal(err)
	}

	assessment := types.RiskAssessment{
		BaseRecord:                  types.BaseRecord{ID: "risk_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:              org.ID,
		ProjectID:                   project.ID,
		ChangeSetID:                 change.ID,
		ServiceID:                   service.ID,
		EnvironmentID:               environment.ID,
		Score:                       67,
		Level:                       types.RiskLevelHigh,
		ConfidenceScore:             0.84,
		Explanation:                 []string{"high-risk change"},
		BlastRadius:                 types.BlastRadius{Scope: "moderate", ServicesImpacted: 2, ResourcesImpacted: 1, Summary: "moderate"},
		RecommendedApprovalLevel:    "platform-owner",
		RecommendedRolloutStrategy:  "canary",
		RecommendedDeploymentWindow: "off-hours-preferred",
		RecommendedGuardrails:       []string{"health-check-gates"},
	}
	if err := store.CreateRiskAssessment(ctx, assessment); err != nil {
		t.Fatal(err)
	}

	plan := types.RolloutPlan{
		BaseRecord:          types.BaseRecord{ID: "roll_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:      org.ID,
		ProjectID:           project.ID,
		ChangeSetID:         change.ID,
		RiskAssessmentID:    assessment.ID,
		Strategy:            "canary",
		ApprovalRequired:    true,
		ApprovalLevel:       "platform-owner",
		DeploymentWindow:    "off-hours-preferred",
		VerificationSignals: []string{"error-rate"},
		RollbackConditions:  []string{"error-rate breach"},
		Guardrails:          []string{"health-check-gates"},
		Steps:               []types.RolloutStep{{Name: "canary", Description: "Deploy to small cohort"}},
		Explanation:         []string{"risk-informed rollout"},
	}
	if err := store.CreateRolloutPlan(ctx, plan); err != nil {
		t.Fatal(err)
	}

	auditEvent := types.AuditEvent{
		BaseRecord:     types.BaseRecord{ID: "audit_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		ActorID:        user.ID,
		ActorType:      "user",
		Actor:          user.Email,
		Action:         "service.registered",
		ResourceType:   "service",
		ResourceID:     service.ID,
		Outcome:        "success",
		Details:        []string{"service registered"},
	}
	if err := store.CreateAuditEvent(ctx, auditEvent); err != nil {
		t.Fatal(err)
	}

	integration := types.Integration{
		BaseRecord:     types.BaseRecord{ID: "int_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		Name:           "GitHub",
		Kind:           "github",
		Mode:           "advisory-ready",
		Status:         "available",
		Capabilities:   []string{"scm"},
		Description:    "GitHub adapter",
	}
	if err := store.UpsertIntegration(ctx, integration); err != nil {
		t.Fatal(err)
	}

	serviceAccount := types.ServiceAccount{
		BaseRecord:      types.BaseRecord{ID: "svcacct_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:  org.ID,
		Name:            "deploy-bot",
		Description:     "deployment bot",
		Role:            "org_member",
		CreatedByUserID: user.ID,
		Status:          "active",
	}
	if err := store.CreateServiceAccount(ctx, serviceAccount); err != nil {
		t.Fatal(err)
	}

	token := types.APIToken{
		BaseRecord:       types.BaseRecord{ID: "token_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:   org.ID,
		ServiceAccountID: serviceAccount.ID,
		Name:             "primary",
		TokenPrefix:      "ccpt_test",
		TokenHash:        "hashed",
		Status:           "active",
	}
	if err := store.CreateAPIToken(ctx, token); err != nil {
		t.Fatal(err)
	}

	repository := types.Repository{
		BaseRecord:     types.BaseRecord{ID: "repo_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		Name:           "checkout",
		Provider:       "github",
		URL:            "https://github.com/acme/checkout",
		DefaultBranch:  "main",
	}
	if err := store.UpsertRepository(ctx, repository); err != nil {
		t.Fatal(err)
	}

	relationship := types.GraphRelationship{
		BaseRecord:          types.BaseRecord{ID: "graph_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:      org.ID,
		ProjectID:           project.ID,
		SourceIntegrationID: integration.ID,
		RelationshipType:    "service_repository",
		FromResourceType:    "service",
		FromResourceID:      service.ID,
		ToResourceType:      "repository",
		ToResourceID:        repository.ID,
		Status:              "active",
		LastObservedAt:      now,
	}
	if err := store.UpsertGraphRelationship(ctx, relationship); err != nil {
		t.Fatal(err)
	}

	execution := types.RolloutExecution{
		BaseRecord:     types.BaseRecord{ID: "exec_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		RolloutPlanID:  plan.ID,
		ChangeSetID:    change.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Status:         "awaiting_approval",
		CurrentStep:    "precheck",
	}
	if err := store.CreateRolloutExecution(ctx, execution); err != nil {
		t.Fatal(err)
	}

	verification := types.VerificationResult{
		BaseRecord:         types.BaseRecord{ID: "verify_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:     org.ID,
		ProjectID:          project.ID,
		RolloutExecutionID: execution.ID,
		RolloutPlanID:      plan.ID,
		ChangeSetID:        change.ID,
		ServiceID:          service.ID,
		EnvironmentID:      environment.ID,
		Status:             "recorded",
		Outcome:            "pass",
		Decision:           "continue",
		Signals:            []string{"latency"},
		Summary:            "healthy rollout",
		Explanation:        []string{"signals within bounds"},
	}
	if err := store.CreateVerificationResult(ctx, verification); err != nil {
		t.Fatal(err)
	}

	organizations, err := store.ListOrganizations(ctx, OrganizationQuery{IDs: []string{org.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(organizations) != 1 {
		t.Fatalf("expected one organization, got %d", len(organizations))
	}

	services, err := store.ListServices(ctx, ServiceQuery{OrganizationID: org.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 1 || services[0].ID != service.ID {
		t.Fatalf("unexpected services payload: %+v", services)
	}

	auditEvents, err := store.ListAuditEvents(ctx, AuditEventQuery{OrganizationID: org.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(auditEvents) != 1 || auditEvents[0].ActorID != user.ID {
		t.Fatalf("unexpected audit events payload: %+v", auditEvents)
	}

	serviceAccounts, err := store.ListServiceAccounts(ctx, ServiceAccountQuery{OrganizationID: org.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(serviceAccounts) != 1 || serviceAccounts[0].ID != serviceAccount.ID {
		t.Fatalf("unexpected service accounts payload: %+v", serviceAccounts)
	}

	tokens, err := store.ListAPITokens(ctx, APITokenQuery{OrganizationID: org.ID, ServiceAccountID: serviceAccount.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].TokenPrefix != token.TokenPrefix {
		t.Fatalf("unexpected api tokens payload: %+v", tokens)
	}

	executions, err := store.ListRolloutExecutions(ctx, RolloutExecutionQuery{OrganizationID: org.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(executions) != 1 || executions[0].ID != execution.ID {
		t.Fatalf("unexpected rollout executions payload: %+v", executions)
	}

	verifications, err := store.ListVerificationResults(ctx, VerificationResultQuery{OrganizationID: org.ID, RolloutExecutionID: execution.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(verifications) != 1 || verifications[0].ID != verification.ID {
		t.Fatalf("unexpected verification payload: %+v", verifications)
	}

	relationships, err := store.ListGraphRelationships(ctx, GraphRelationshipQuery{OrganizationID: org.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(relationships) != 1 || relationships[0].ID != relationship.ID {
		t.Fatalf("unexpected graph relationships payload: %+v", relationships)
	}
}

func truncateTables(db *sql.DB) error {
	_, err := db.Exec(`
		TRUNCATE TABLE
			verification_results,
			rollout_executions,
			graph_relationships,
			api_tokens,
			service_accounts,
			project_memberships,
			organization_memberships,
			audit_events,
			rollout_plans,
			risk_assessments,
			change_artifacts,
			change_sets,
			integrations,
			repositories,
			environments,
			services,
			teams,
			projects,
			users,
			organizations
		RESTART IDENTITY CASCADE
	`)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

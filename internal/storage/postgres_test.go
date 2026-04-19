package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
		BaseRecord: types.BaseRecord{
			ID:        "repo_test_pg",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata: types.Metadata{
				"ownership": map[string]any{
					"status": "imported",
					"source": "codeowners_import",
				},
			},
		},
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

	resource := types.DiscoveredResource{
		BaseRecord: types.BaseRecord{
			ID:        "resource_test_pg",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata: types.Metadata{
				"inferred_owner": map[string]any{
					"team_id": team.ID,
					"source":  "service_mapping",
				},
			},
		},
		OrganizationID: org.ID,
		IntegrationID:  integration.ID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		RepositoryID:   repository.ID,
		ResourceType:   "kubernetes_workload",
		Provider:       "kubernetes",
		ExternalID:     "prod/checkout",
		Namespace:      "prod",
		Name:           "checkout",
		Status:         "mapped",
		Health:         "healthy",
		Summary:        "all replicas available",
		LastSeenAt:     &now,
	}
	if err := store.UpsertDiscoveredResource(ctx, resource); err != nil {
		t.Fatal(err)
	}

	relationship := types.GraphRelationship{
		BaseRecord: types.BaseRecord{
			ID:        "graph_test_pg",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata: types.Metadata{
				"provenance_source": "manual",
			},
		},
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
		BaseRecord:      types.BaseRecord{ID: "exec_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:  org.ID,
		ProjectID:       project.ID,
		RolloutPlanID:   plan.ID,
		ChangeSetID:     change.ID,
		ServiceID:       service.ID,
		EnvironmentID:   environment.ID,
		BackendType:     "simulated",
		BackendStatus:   "progressing",
		ProgressPercent: 60,
		Status:          "awaiting_approval",
		CurrentStep:     "precheck",
	}
	if err := store.CreateRolloutExecution(ctx, execution); err != nil {
		t.Fatal(err)
	}

	snapshot := types.SignalSnapshot{
		BaseRecord:         types.BaseRecord{ID: "signal_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:     org.ID,
		ProjectID:          project.ID,
		RolloutExecutionID: execution.ID,
		RolloutPlanID:      plan.ID,
		ChangeSetID:        change.ID,
		ServiceID:          service.ID,
		EnvironmentID:      environment.ID,
		ProviderType:       "simulated",
		Health:             "healthy",
		Summary:            "runtime remained healthy",
		Signals:            []types.SignalValue{{Name: "latency", Category: "technical", Value: 120, Status: "healthy"}},
		WindowStart:        now.Add(-5 * time.Minute),
		WindowEnd:          now,
	}
	if err := store.CreateSignalSnapshot(ctx, snapshot); err != nil {
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
		Decision:           "verified",
		Signals:            []string{"latency"},
		Automated:          true,
		DecisionSource:     "control_loop",
		SignalSnapshotIDs:  []string{snapshot.ID},
		Summary:            "healthy rollout",
		Explanation:        []string{"signals within bounds"},
	}
	if err := store.CreateVerificationResult(ctx, verification); err != nil {
		t.Fatal(err)
	}

	rollbackPolicy := types.RollbackPolicy{
		BaseRecord:                types.BaseRecord{ID: "rpol_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:            org.ID,
		ProjectID:                 project.ID,
		ServiceID:                 service.ID,
		EnvironmentID:             environment.ID,
		Name:                      "Prod strict",
		Enabled:                   true,
		Priority:                  5,
		MaxErrorRate:              1,
		MaxLatencyMs:              500,
		MaxUnhealthyInstances:     0,
		MaxVerificationFailures:   1,
		RollbackOnProviderFailure: true,
		RollbackOnCriticalSignals: true,
	}
	if err := store.CreateRollbackPolicy(ctx, rollbackPolicy); err != nil {
		t.Fatal(err)
	}

	policy := types.Policy{
		BaseRecord: types.BaseRecord{
			ID:        "pol_test_pg",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata: types.Metadata{
				"source": "postgres-round-trip",
			},
		},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Name:           "Production High Risk Review",
		Code:           "production-high-risk-review",
		Scope:          "environment",
		AppliesTo:      "rollout_plan",
		Mode:           "require_manual_review",
		Enabled:        true,
		Priority:       120,
		Description:    "Require manual review for high-risk production rollout planning.",
		Conditions: types.PolicyCondition{
			MinRiskLevel:   string(types.RiskLevelHigh),
			ProductionOnly: true,
		},
		Triggers: []string{"risk>=high", "environment=production"},
	}
	if err := store.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	policy.Enabled = false
	policy.Description = "Disable manual review override for round-trip coverage."
	policy.UpdatedAt = now.Add(30 * time.Second)
	if err := store.UpdatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}

	policyDecision := types.PolicyDecision{
		BaseRecord: types.BaseRecord{
			ID:        "pdec_test_pg",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata: types.Metadata{
				"evaluation_outcome": "evaluated",
			},
		},
		OrganizationID:   org.ID,
		ProjectID:        project.ID,
		ServiceID:        service.ID,
		EnvironmentID:    environment.ID,
		PolicyID:         policy.ID,
		PolicyName:       policy.Name,
		PolicyCode:       policy.Code,
		PolicyScope:      policy.Scope,
		AppliesTo:        policy.AppliesTo,
		Mode:             policy.Mode,
		ChangeSetID:      change.ID,
		RiskAssessmentID: assessment.ID,
		RolloutPlanID:    plan.ID,
		Outcome:          policy.Mode,
		Summary:          "Production High Risk Review requires manual review when risk level high meets minimum high.",
		Reasons:          []string{"environment is production", "risk level high meets minimum high"},
	}
	if err := store.CreatePolicyDecision(ctx, policyDecision); err != nil {
		t.Fatal(err)
	}

	statusEvent := types.StatusEvent{
		BaseRecord:         types.BaseRecord{ID: "status_test_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:     org.ID,
		ProjectID:          project.ID,
		ServiceID:          service.ID,
		EnvironmentID:      environment.ID,
		RolloutExecutionID: execution.ID,
		ChangeSetID:        change.ID,
		ResourceType:       "rollout_execution",
		ResourceID:         execution.ID,
		EventType:          "rollout.execution.transitioned",
		Category:           "rollout",
		Severity:           "info",
		PreviousState:      "planned",
		NewState:           "in_progress",
		Outcome:            "success",
		ActorID:            user.ID,
		ActorType:          "user",
		Actor:              user.Email,
		Source:             "api",
		Summary:            "rollout started",
		Explanation:        []string{"operator started rollout"},
	}
	if err := store.CreateStatusEvent(ctx, statusEvent); err != nil {
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

	discoveredResources, err := store.ListDiscoveredResources(ctx, DiscoveredResourceQuery{OrganizationID: org.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(discoveredResources) != 1 || discoveredResources[0].Metadata["inferred_owner"] == nil {
		t.Fatalf("expected discovered resource provenance to persist, got %+v", discoveredResources)
	}

	executions, err := store.ListRolloutExecutions(ctx, RolloutExecutionQuery{OrganizationID: org.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(executions) != 1 || executions[0].ID != execution.ID {
		t.Fatalf("unexpected rollout executions payload: %+v", executions)
	}
	if executions[0].BackendType != "simulated" || executions[0].ProgressPercent != 60 {
		t.Fatalf("expected runtime fields to persist, got %+v", executions[0])
	}

	verifications, err := store.ListVerificationResults(ctx, VerificationResultQuery{OrganizationID: org.ID, RolloutExecutionID: execution.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(verifications) != 1 || verifications[0].ID != verification.ID {
		t.Fatalf("unexpected verification payload: %+v", verifications)
	}
	if !verifications[0].Automated || verifications[0].DecisionSource != "control_loop" {
		t.Fatalf("expected verification automation fields to persist, got %+v", verifications[0])
	}

	snapshots, err := store.ListSignalSnapshots(ctx, SignalSnapshotQuery{OrganizationID: org.ID, RolloutExecutionID: execution.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || snapshots[0].ID != snapshot.ID {
		t.Fatalf("unexpected signal snapshot payload: %+v", snapshots)
	}

	relationships, err := store.ListGraphRelationships(ctx, GraphRelationshipQuery{OrganizationID: org.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(relationships) != 1 || relationships[0].ID != relationship.ID {
		t.Fatalf("unexpected graph relationships payload: %+v", relationships)
	}
	if relationships[0].Metadata["provenance_source"] != "manual" {
		t.Fatalf("expected graph relationship metadata to persist, got %+v", relationships[0])
	}

	rollbackPolicies, err := store.ListRollbackPolicies(ctx, RollbackPolicyQuery{OrganizationID: org.ID, ProjectID: project.ID, EnabledOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(rollbackPolicies) != 1 || rollbackPolicies[0].ID != rollbackPolicy.ID {
		t.Fatalf("unexpected rollback policies payload: %+v", rollbackPolicies)
	}

	policies, err := store.ListPolicies(ctx, PolicyQuery{OrganizationID: org.ID, AppliesTo: "rollout_plan"})
	if err != nil {
		t.Fatal(err)
	}
	if len(policies) != 1 || policies[0].ID != policy.ID {
		t.Fatalf("unexpected policies payload: %+v", policies)
	}
	storedPolicy, err := store.GetPolicy(ctx, policy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedPolicy.Enabled || storedPolicy.Description != policy.Description || storedPolicy.Scope != "environment" {
		t.Fatalf("expected policy round-trip and update persistence, got %+v", storedPolicy)
	}

	policyDecisions, err := store.ListPolicyDecisions(ctx, PolicyDecisionQuery{
		OrganizationID:   org.ID,
		PolicyID:         policy.ID,
		RiskAssessmentID: assessment.ID,
		AppliesTo:        "rollout_plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(policyDecisions) != 1 || policyDecisions[0].ID != policyDecision.ID {
		t.Fatalf("unexpected policy decisions payload: %+v", policyDecisions)
	}
	if policyDecisions[0].PolicyCode != policy.Code || policyDecisions[0].Outcome != "require_manual_review" {
		t.Fatalf("expected policy decision round-trip to preserve code and outcome, got %+v", policyDecisions[0])
	}

	statusEvents, err := store.ListStatusEvents(ctx, StatusEventQuery{OrganizationID: org.ID, RollbackOnly: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(statusEvents) != 1 || statusEvents[0].ID != statusEvent.ID {
		t.Fatalf("unexpected status events payload: %+v", statusEvents)
	}
}

func TestPostgresStoreStatusEventFiltersAndNotFound(t *testing.T) {
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
		BaseRecord: types.BaseRecord{ID: "org_status_pg", CreatedAt: now, UpdatedAt: now},
		Name:       "Acme",
		Slug:       "acme-status-pg",
		Tier:       "growth",
		Mode:       "startup",
	}
	if err := store.CreateOrganization(ctx, org); err != nil {
		t.Fatal(err)
	}
	project := types.Project{
		BaseRecord:     types.BaseRecord{ID: "proj_status_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		Name:           "Platform",
		Slug:           "platform",
		Status:         "active",
	}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	team := types.Team{
		BaseRecord:     types.BaseRecord{ID: "team_status_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		Name:           "Platform Team",
		Slug:           "platform-team",
		Status:         "active",
	}
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatal(err)
	}
	service := types.Service{
		BaseRecord:     types.BaseRecord{ID: "svc_status_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
		Criticality:    "high",
		Status:         "active",
	}
	if err := store.CreateService(ctx, service); err != nil {
		t.Fatal(err)
	}
	environment := types.Environment{
		BaseRecord:     types.BaseRecord{ID: "env_status_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
		Status:         "active",
	}
	if err := store.CreateEnvironment(ctx, environment); err != nil {
		t.Fatal(err)
	}

	rollbackEvent := types.StatusEvent{
		BaseRecord:         types.BaseRecord{ID: "status_rollback_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:     org.ID,
		ProjectID:          project.ID,
		ServiceID:          service.ID,
		EnvironmentID:      environment.ID,
		RolloutExecutionID: "exec_rollback_pg",
		ResourceType:       "rollout_execution",
		ResourceID:         "exec_rollback_pg",
		EventType:          "rollout.execution.rollback",
		Category:           "rollout",
		Severity:           "critical",
		PreviousState:      "in_progress",
		NewState:           "rolled_back",
		Outcome:            "success",
		ActorID:            "svcacct_test",
		ActorType:          "service_account",
		Actor:              "control-loop",
		Source:             "control_loop",
		Automated:          true,
		Summary:            "rollback triggered after critical error rate breach",
		Explanation:        []string{"rollback", "error rate"},
	}
	if err := store.CreateStatusEvent(ctx, rollbackEvent); err != nil {
		t.Fatal(err)
	}

	verificationEvent := types.StatusEvent{
		BaseRecord:         types.BaseRecord{ID: "status_verify_pg", CreatedAt: now.Add(1 * time.Minute), UpdatedAt: now.Add(1 * time.Minute)},
		OrganizationID:     org.ID,
		ProjectID:          project.ID,
		ServiceID:          service.ID,
		EnvironmentID:      environment.ID,
		RolloutExecutionID: "exec_rollback_pg",
		ResourceType:       "verification_result",
		ResourceID:         "verify_pg",
		EventType:          "verification.recorded",
		Category:           "verification",
		Severity:           "info",
		Outcome:            "recorded",
		ActorID:            "user_test",
		ActorType:          "user",
		Actor:              "owner@acme.local",
		Source:             "api",
		Summary:            "healthy verification recorded",
		Explanation:        []string{"healthy"},
	}
	if err := store.CreateStatusEvent(ctx, verificationEvent); err != nil {
		t.Fatal(err)
	}

	rollbackOnly, err := store.ListStatusEvents(ctx, StatusEventQuery{
		OrganizationID: org.ID,
		RollbackOnly:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rollbackOnly) != 1 || rollbackOnly[0].ID != rollbackEvent.ID {
		t.Fatalf("unexpected rollback-only payload: %+v", rollbackOnly)
	}

	searchResults, err := store.ListStatusEvents(ctx, StatusEventQuery{
		OrganizationID: org.ID,
		Search:         "critical error rate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(searchResults) != 1 || searchResults[0].ID != rollbackEvent.ID {
		t.Fatalf("unexpected search payload: %+v", searchResults)
	}

	serviceResults, err := store.ListStatusEvents(ctx, StatusEventQuery{
		OrganizationID: org.ID,
		ServiceID:      service.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(serviceResults) != 2 {
		t.Fatalf("expected two service-scoped events, got %+v", serviceResults)
	}

	rollbackCount, err := store.CountStatusEvents(ctx, StatusEventQuery{
		OrganizationID:     org.ID,
		RolloutExecutionID: "exec_rollback_pg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rollbackCount != 2 {
		t.Fatalf("expected two rollout-scoped status events, got %d", rollbackCount)
	}

	if _, err := store.GetStatusEvent(ctx, "missing-status"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostgresStoreUpdateOutboxEventIfStatusHonorsExpectedStatus(t *testing.T) {
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
	event := types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        "evt_pg_compare_and_update",
			CreatedAt: now.Add(-2 * time.Minute),
			UpdatedAt: now.Add(-1 * time.Minute),
			Metadata: types.Metadata{
				"last_error_class": "temporary",
			},
		},
		EventType:      "status.created",
		OrganizationID: "org_pg_compare",
		ResourceType:   "status_event",
		ResourceID:     "status_pg_compare",
		Status:         "error",
		Attempts:       2,
		LastError:      "temporary dispatch failure",
		NextAttemptAt:  ptrTime(now.Add(10 * time.Minute)),
	}
	if err := store.CreateOutboxEvent(ctx, event); err != nil {
		t.Fatal(err)
	}

	recovered := event
	recovered.Status = "pending"
	recovered.NextAttemptAt = nil
	recovered.ClaimedAt = nil
	recovered.ProcessedAt = nil
	recovered.UpdatedAt = now

	ok, err := store.UpdateOutboxEventIfStatus(ctx, recovered, "error")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected compare-and-update to succeed for matching status")
	}

	stored, err := store.GetOutboxEvent(ctx, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "pending" || stored.NextAttemptAt != nil {
		t.Fatalf("expected stored event to reflect successful status-guarded update, got %+v", stored)
	}

	stored.Status = "processing"
	stored.ClaimedAt = ptrTime(now.Add(5 * time.Second))
	stored.UpdatedAt = now.Add(5 * time.Second)
	if err := store.UpdateOutboxEvent(ctx, stored); err != nil {
		t.Fatal(err)
	}

	repeated := stored
	repeated.Status = "pending"
	repeated.ClaimedAt = nil
	repeated.UpdatedAt = now.Add(10 * time.Second)
	ok, err = store.UpdateOutboxEventIfStatus(ctx, repeated, "error")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("expected compare-and-update to reject stale expected status")
	}

	afterMismatch, err := store.GetOutboxEvent(ctx, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterMismatch.Status != "processing" || afterMismatch.ClaimedAt == nil {
		t.Fatalf("expected failed compare-and-update to preserve fresher processing state, got %+v", afterMismatch)
	}
}

func TestPostgresStoreBrowserSessionRoundTrip(t *testing.T) {
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
		BaseRecord: types.BaseRecord{ID: "org_browser_pg", CreatedAt: now, UpdatedAt: now},
		Name:       "Browser Sessions",
		Slug:       "browser-sessions",
		Tier:       "growth",
		Mode:       "startup",
	}
	if err := store.CreateOrganization(ctx, org); err != nil {
		t.Fatal(err)
	}
	user := types.User{
		BaseRecord:     types.BaseRecord{ID: "user_browser_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: org.ID,
		Email:          "owner@browser.local",
		DisplayName:    "Browser Owner",
		Status:         "active",
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	expiresAt := now.Add(2 * time.Hour)
	session := types.BrowserSession{
		BaseRecord: types.BaseRecord{
			ID:        "sess_browser_pg",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata: types.Metadata{
				"issued_by": "postgres-test",
			},
		},
		UserID:         user.ID,
		SessionHash:    "hash_browser_pg",
		AuthMethod:     "password",
		AuthProviderID: "provider_browser_pg",
		AuthProvider:   "Browser Auth",
		LastSeenAt:     ptrTime(now),
		ExpiresAt:      expiresAt,
	}
	if err := store.CreateBrowserSession(ctx, session); err != nil {
		t.Fatal(err)
	}

	stored, err := store.GetBrowserSessionByHash(ctx, session.SessionHash)
	if err != nil {
		t.Fatal(err)
	}
	if stored.UserID != user.ID || stored.AuthMethod != "password" {
		t.Fatalf("expected browser session round-trip, got %+v", stored)
	}

	revokedAt := now.Add(30 * time.Minute)
	stored.RevokedAt = &revokedAt
	stored.UpdatedAt = now.Add(31 * time.Minute)
	if err := store.UpdateBrowserSession(ctx, stored); err != nil {
		t.Fatal(err)
	}

	updated, err := store.GetBrowserSessionByHash(ctx, session.SessionHash)
	if err != nil {
		t.Fatal(err)
	}
	if updated.RevokedAt == nil || !updated.RevokedAt.Equal(revokedAt) {
		t.Fatalf("expected browser session revocation to persist, got %+v", updated)
	}
}

func TestPostgresStoreWithinTransactionRollsBackOnError(t *testing.T) {
	dsn := postgresTestDSN()

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
	rollbackOrg := types.Organization{
		BaseRecord: types.BaseRecord{ID: "org_tx_rollback_pg", CreatedAt: now, UpdatedAt: now},
		Name:       "Rollback Org",
		Slug:       "rollback-org-pg",
		Tier:       "growth",
		Mode:       "startup",
	}

	expectedErr := errors.New("force rollback")
	err = store.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := store.CreateOrganization(txCtx, rollbackOrg); err != nil {
			return err
		}
		if _, err := store.GetOrganization(txCtx, rollbackOrg.ID); err != nil {
			return err
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected transaction error %v, got %v", expectedErr, err)
	}
	if _, err := store.GetOrganization(ctx, rollbackOrg.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected rollback organization to be absent after rollback, got %v", err)
	}

	commitOrg := types.Organization{
		BaseRecord: types.BaseRecord{ID: "org_tx_commit_pg", CreatedAt: now.Add(1 * time.Minute), UpdatedAt: now.Add(1 * time.Minute)},
		Name:       "Commit Org",
		Slug:       "commit-org-pg",
		Tier:       "growth",
		Mode:       "startup",
	}
	if err := store.WithinTransaction(ctx, func(txCtx context.Context) error {
		return store.CreateOrganization(txCtx, commitOrg)
	}); err != nil {
		t.Fatalf("expected transaction commit to succeed, got %v", err)
	}
	stored, err := store.GetOrganization(ctx, commitOrg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Slug != commitOrg.Slug {
		t.Fatalf("expected committed organization to persist, got %+v", stored)
	}
}

func TestPostgresStoreDuplicateHandlingAndPagination(t *testing.T) {
	dsn := postgresTestDSN()

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
	orgs := []types.Organization{
		{
			BaseRecord: types.BaseRecord{ID: "org_page_1_pg", CreatedAt: now, UpdatedAt: now},
			Name:       "Alpha",
			Slug:       "alpha-pg",
			Tier:       "growth",
			Mode:       "startup",
		},
		{
			BaseRecord: types.BaseRecord{ID: "org_page_2_pg", CreatedAt: now.Add(1 * time.Minute), UpdatedAt: now.Add(1 * time.Minute)},
			Name:       "Bravo",
			Slug:       "bravo-pg",
			Tier:       "growth",
			Mode:       "startup",
		},
		{
			BaseRecord: types.BaseRecord{ID: "org_page_3_pg", CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute)},
			Name:       "Charlie",
			Slug:       "charlie-pg",
			Tier:       "growth",
			Mode:       "startup",
		},
	}
	for _, org := range orgs {
		if err := store.CreateOrganization(ctx, org); err != nil {
			t.Fatal(err)
		}
	}

	paged, err := store.ListOrganizations(ctx, OrganizationQuery{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(paged) != 2 || paged[0].ID != orgs[1].ID || paged[1].ID != orgs[2].ID {
		t.Fatalf("expected ordered paginated organizations [org_page_2_pg org_page_3_pg], got %+v", paged)
	}

	duplicateOrg := types.Organization{
		BaseRecord: types.BaseRecord{ID: "org_duplicate_pg", CreatedAt: now.Add(3 * time.Minute), UpdatedAt: now.Add(3 * time.Minute)},
		Name:       "Duplicate",
		Slug:       orgs[0].Slug,
		Tier:       "growth",
		Mode:       "startup",
	}
	if err := store.CreateOrganization(ctx, duplicateOrg); err == nil {
		t.Fatal("expected duplicate organization slug create to fail")
	}

	serviceAccount := types.ServiceAccount{
		BaseRecord:     types.BaseRecord{ID: "svcacct_duplicate_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: orgs[0].ID,
		Name:           "deploy-bot",
		Role:           "org_member",
		Status:         "active",
	}
	if err := store.CreateServiceAccount(ctx, serviceAccount); err != nil {
		t.Fatal(err)
	}

	token := types.APIToken{
		BaseRecord:       types.BaseRecord{ID: "token_duplicate_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID:   orgs[0].ID,
		ServiceAccountID: serviceAccount.ID,
		Name:             "primary",
		TokenPrefix:      "ccpt_duplicate_pg",
		TokenHash:        "hash-primary",
		Status:           "active",
	}
	if err := store.CreateAPIToken(ctx, token); err != nil {
		t.Fatal(err)
	}

	duplicateToken := token
	duplicateToken.ID = "token_duplicate_pg_2"
	duplicateToken.Name = "secondary"
	duplicateToken.TokenHash = "hash-secondary"
	if err := store.CreateAPIToken(ctx, duplicateToken); err == nil {
		t.Fatal("expected duplicate token prefix create to fail")
	}

	repository := types.Repository{
		BaseRecord:     types.BaseRecord{ID: "repo_duplicate_pg", CreatedAt: now, UpdatedAt: now},
		OrganizationID: orgs[0].ID,
		Name:           "checkout",
		Provider:       "github",
		URL:            "https://github.com/acme/checkout",
		DefaultBranch:  "main",
		Status:         "mapped",
	}
	if err := store.UpsertRepository(ctx, repository); err != nil {
		t.Fatal(err)
	}

	duplicateRepository := repository
	duplicateRepository.ID = "repo_duplicate_pg_2"
	duplicateRepository.Name = "checkout-clone"
	duplicateRepository.CreatedAt = now.Add(4 * time.Minute)
	duplicateRepository.UpdatedAt = now.Add(4 * time.Minute)
	if err := store.UpsertRepository(ctx, duplicateRepository); err == nil {
		t.Fatal("expected duplicate organization/url repository upsert to fail")
	}
}

func TestPostgresStoreFreshBootstrapAppliesMigrations(t *testing.T) {
	dsn := postgresTestDSN()
	freshDSN, cleanup, err := createTemporaryDatabase(dsn)
	if err != nil {
		t.Skipf("temporary database unavailable: %v", err)
	}
	defer cleanup()

	cfg := common.LoadConfig()
	cfg.DBDSN = freshDSN
	cfg.AutoMigrate = true

	store, err := NewPostgresStore(cfg)
	if err != nil {
		t.Skipf("fresh bootstrap postgres unavailable: %v", err)
	}
	defer store.Close()

	expectedMigrations := countMigrationFiles(t)
	var appliedCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&appliedCount); err != nil {
		t.Fatal(err)
	}
	if appliedCount != expectedMigrations {
		t.Fatalf("expected %d applied migrations on fresh bootstrap, got %d", expectedMigrations, appliedCount)
	}

	if err := ApplyMigrations(context.Background(), store.db, filepath.Join("db", "migrations")); err != nil {
		t.Fatalf("expected migration replay to be idempotent, got %v", err)
	}
	var replayedCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&replayedCount); err != nil {
		t.Fatal(err)
	}
	if replayedCount != expectedMigrations {
		t.Fatalf("expected %d applied migrations after replay, got %d", expectedMigrations, replayedCount)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	org := types.Organization{
		BaseRecord: types.BaseRecord{ID: "org_bootstrap_pg", CreatedAt: now, UpdatedAt: now},
		Name:       "Bootstrap Org",
		Slug:       "bootstrap-org-pg",
		Tier:       "growth",
		Mode:       "startup",
	}
	if err := store.CreateOrganization(ctx, org); err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetOrganization(ctx, org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID != org.ID || stored.Slug != org.Slug {
		t.Fatalf("expected fresh bootstrap database to accept persisted writes, got %+v", stored)
	}
}

func truncateTables(db *sql.DB) error {
	_, err := db.Exec(`
		TRUNCATE TABLE
			browser_sessions,
			outbox_events,
			verification_results,
			status_events,
			rollout_executions,
			signal_snapshots,
			rollback_policies,
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

func ptrTime(value time.Time) *time.Time {
	return &value
}

func postgresTestDSN() string {
	dsn := os.Getenv("CCP_TEST_DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("CCP_DB_DSN")
	}
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/change_control_plane?sslmode=disable"
	}
	return dsn
}

func countMigrationFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join("db", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count
}

func createTemporaryDatabase(sourceDSN string) (string, func(), error) {
	adminDB, err := sql.Open("pgx", sourceDSN)
	if err != nil {
		return "", nil, err
	}

	cleanupDB := func() {
		_ = adminDB.Close()
	}

	if err := adminDB.Ping(); err != nil {
		cleanupDB()
		return "", nil, err
	}

	parsed, err := url.Parse(sourceDSN)
	if err != nil {
		cleanupDB()
		return "", nil, err
	}

	dbName := fmt.Sprintf("ccp_bootstrap_%d", time.Now().UTC().UnixNano())
	if _, err := adminDB.Exec(`CREATE DATABASE ` + dbName); err != nil {
		cleanupDB()
		return "", nil, err
	}

	fresh := *parsed
	fresh.Path = "/" + dbName
	fresh.RawPath = ""

	cleanup := func() {
		_, _ = adminDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName)
		_, _ = adminDB.Exec(`DROP DATABASE ` + dbName)
		cleanupDB()
	}

	return fresh.String(), cleanup, nil
}

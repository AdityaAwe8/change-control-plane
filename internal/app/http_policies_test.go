package app_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestPolicyCRUDAndAdvisoryEvaluationAPI(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-policy@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Policy",
		OrganizationSlug: "acme-policy",
	})
	orgID := admin.Session.ActiveOrganizationID

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: orgID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, orgID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, orgID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   orgID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "medium",
		HasSLO:           true,
		HasObservability: false,
	}, admin.Token, orgID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Staging",
		Slug:           "staging",
		Type:           "staging",
		Region:         "us-central1",
	}, admin.Token, orgID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "policy advisory validation",
		ChangeTypes:    []string{"code"},
		FileCount:      3,
		ResourceCount:  1,
	}, admin.Token, orgID)

	policy := postItemAuth[types.Policy](t, server.URL+"/api/v1/policies", types.CreatePolicyRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		Name:           "Observability Coverage Review",
		Code:           "observability-coverage-review",
		AppliesTo:      "risk_assessment",
		Mode:           "advisory",
		Priority:       70,
		Description:    "Warn when a service lacks observability coverage before risk review.",
		Conditions: types.PolicyCondition{
			MissingCapabilities: []string{"observability"},
		},
	}, admin.Token, orgID)
	if policy.Scope != "service" || policy.AppliesTo != "risk_assessment" || policy.Mode != "advisory" {
		t.Fatalf("unexpected created policy scope or mode: %+v", policy)
	}
	if !strings.Contains(strings.Join(policy.Triggers, ","), "missing=observability") {
		t.Fatalf("expected computed trigger for missing observability, got %+v", policy.Triggers)
	}

	policies := getListAuth[types.Policy](t, server.URL+"/api/v1/policies", admin.Token, orgID, http.StatusOK)
	if !containsPolicyID(policies, policy.ID) {
		t.Fatalf("expected policy list to include %s, got %+v", policy.ID, policies)
	}

	fetched := getItemAuth[types.Policy](t, server.URL+"/api/v1/policies/"+policy.ID, admin.Token, orgID, http.StatusOK)
	if fetched.ID != policy.ID || fetched.Code != policy.Code {
		t.Fatalf("expected policy show to round-trip created policy, got %+v", fetched)
	}

	assessment := postItemAuth[types.RiskAssessmentResult](t, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: change.ID,
	}, admin.Token, orgID)
	if !containsPolicyDecisionCode(assessment.PolicyDecisions, policy.Code) {
		t.Fatalf("expected risk assessment to include advisory decision for %s, got %+v", policy.Code, assessment.PolicyDecisions)
	}

	decisions := getListAuth[types.PolicyDecision](t, server.URL+"/api/v1/policy-decisions?risk_assessment_id="+assessment.Assessment.ID+"&policy_id="+policy.ID, admin.Token, orgID, http.StatusOK)
	if len(decisions) != 1 {
		t.Fatalf("expected one policy decision for created policy, got %+v", decisions)
	}
	if decisions[0].Outcome != "advisory" || decisions[0].PolicyCode != policy.Code || len(decisions[0].Reasons) == 0 {
		t.Fatalf("expected persisted advisory decision with explanation, got %+v", decisions[0])
	}

	updatedDescription := "Warn when observability evidence is missing before risk review."
	updatedPriority := 99
	disabled := false
	updated := patchItemAuth[types.Policy](t, server.URL+"/api/v1/policies/"+policy.ID, types.UpdatePolicyRequest{
		Description: &updatedDescription,
		Priority:    &updatedPriority,
		Enabled:     &disabled,
	}, admin.Token, orgID, http.StatusOK)
	if updated.Description != updatedDescription || updated.Priority != updatedPriority || updated.Enabled {
		t.Fatalf("expected policy update to persist description, priority, and enabled=false, got %+v", updated)
	}

	secondAssessment := postItemAuth[types.RiskAssessmentResult](t, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: change.ID,
	}, admin.Token, orgID)
	if containsPolicyDecisionCode(secondAssessment.PolicyDecisions, policy.Code) {
		t.Fatalf("expected disabled policy %s to stop matching, got %+v", policy.Code, secondAssessment.PolicyDecisions)
	}

	auditEvents := getListAuth[types.AuditEvent](t, server.URL+"/api/v1/audit-events", admin.Token, orgID, http.StatusOK)
	if !hasAuditEventAction(auditEvents, "policy.created", policy.ID) || !hasAuditEventAction(auditEvents, "policy.updated", policy.ID) {
		t.Fatalf("expected audit evidence for policy create/update, got %+v", auditEvents)
	}

	statusEvents := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/status-events?search=policy&limit=25", admin.Token, orgID, http.StatusOK)
	if !hasStatusEventType(statusEvents, "policy.created") || !hasStatusEventType(statusEvents, "policy.updated") {
		t.Fatalf("expected status evidence for policy create/update, got %+v", statusEvents)
	}
}

func TestPolicyRolloutReviewAndBlockOutcomes(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-governance@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Governance",
		OrganizationSlug: "acme-governance",
	})
	orgID := admin.Session.ActiveOrganizationID

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: orgID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, orgID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, orgID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   orgID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "low",
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, orgID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, orgID)

	reviewPolicy := postItemAuth[types.Policy](t, server.URL+"/api/v1/policies", types.CreatePolicyRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Name:           "Production High Risk Review",
		Code:           "production-high-risk-review",
		AppliesTo:      "rollout_plan",
		Mode:           "require_manual_review",
		Priority:       120,
		Description:    "Force policy review for production rollout planning.",
		Conditions: types.PolicyCondition{
			ProductionOnly: true,
		},
	}, admin.Token, orgID)

	highChange := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "review-gated rollout",
		ChangeTypes:    []string{},
	}, admin.Token, orgID)
	reviewPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: highChange.ID,
	}, admin.Token, orgID)
	if reviewPlan.Plan.ApprovalLevel != "policy-review" {
		t.Fatalf("expected rollout plan approval level to be policy-review, got %+v", reviewPlan.Plan)
	}
	if !containsPolicyDecisionCode(reviewPlan.PolicyDecisions, reviewPolicy.Code) {
		t.Fatalf("expected rollout plan result to include manual-review decision for %s, got %+v", reviewPolicy.Code, reviewPlan.PolicyDecisions)
	}

	reviewDecisions := getListAuth[types.PolicyDecision](t, server.URL+"/api/v1/policy-decisions?rollout_plan_id="+reviewPlan.Plan.ID+"&policy_id="+reviewPolicy.ID, admin.Token, orgID, http.StatusOK)
	if len(reviewDecisions) != 1 || reviewDecisions[0].Outcome != "require_manual_review" {
		t.Fatalf("expected persisted manual-review decision for rollout plan, got %+v", reviewDecisions)
	}

	blockPolicy := postItemAuth[types.Policy](t, server.URL+"/api/v1/policies", types.CreatePolicyRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Name:           "Critical Schema Freeze",
		Code:           "critical-schema-freeze",
		AppliesTo:      "rollout_plan",
		Mode:           "block",
		Priority:       140,
		Description:    "Block rollout planning for critical production schema changes.",
		Conditions: types.PolicyCondition{
			ProductionOnly:  true,
			MinRiskLevel:    "critical",
			RequiredTouches: []string{"schema"},
		},
	}, admin.Token, orgID)

	blockChange := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID:        orgID,
		ProjectID:             project.ID,
		ServiceID:             service.ID,
		EnvironmentID:         environment.ID,
		Summary:               "blocked schema rollout",
		ChangeTypes:           []string{"schema", "infra", "iam", "dependency"},
		FileCount:             20,
		ResourceCount:         4,
		TouchesInfrastructure: true,
		TouchesIAM:            true,
		TouchesSchema:         true,
		DependencyChanges:     true,
	}, admin.Token, orgID)

	status, body := requestStatusAndBody(t, http.MethodPost, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: blockChange.ID,
	}, admin.Token, orgID)
	if status != http.StatusBadRequest {
		t.Fatalf("expected rollout plan block to return 400, got %d (%s)", status, string(body))
	}
	if !strings.Contains(string(body), "rollout plan blocked by policy") {
		t.Fatalf("expected block response to mention policy block, got %s", string(body))
	}

	blockDecisions := getListAuth[types.PolicyDecision](t, server.URL+"/api/v1/policy-decisions?change_set_id="+blockChange.ID+"&applies_to=rollout_plan", admin.Token, orgID, http.StatusOK)
	if !containsPolicyDecisionCode(blockDecisions, blockPolicy.Code) {
		t.Fatalf("expected persisted blocked rollout-plan decision for %s, got %+v", blockPolicy.Code, blockDecisions)
	}
	blockedDecision := decisionByCode(blockDecisions, blockPolicy.Code)
	if blockedDecision.Outcome != "block" || blockedDecision.Metadata["blocked_attempt"] != true {
		t.Fatalf("expected blocked decision metadata to preserve blocked attempt evidence, got %+v", blockedDecision)
	}

	auditEvents := getListAuth[types.AuditEvent](t, server.URL+"/api/v1/audit-events", admin.Token, orgID, http.StatusOK)
	if !hasAuditEventAction(auditEvents, "policy.blocked", blockChange.ID) {
		t.Fatalf("expected audit evidence for blocked rollout planning, got %+v", auditEvents)
	}

	statusEvents := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/status-events?search=policy&service_id="+service.ID+"&limit=25", admin.Token, orgID, http.StatusOK)
	if !hasStatusEventType(statusEvents, "policy.blocked") {
		t.Fatalf("expected status evidence for blocked rollout planning, got %+v", statusEvents)
	}
}

func TestPolicyRoutesEnforceRBACAndTenantScope(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-policy-rbac@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Policy RBAC",
		OrganizationSlug: "acme-policy-rbac",
	})
	member := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "member-policy-rbac@acme.local",
		DisplayName:      "Member",
		OrganizationSlug: "acme-policy-rbac",
		Roles:            []string{"org_member"},
	})
	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-policy-other@acme.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Org",
		OrganizationSlug: "other-policy-org",
	})
	orgID := admin.Session.ActiveOrganizationID

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: orgID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, orgID)

	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/policies", types.CreatePolicyRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Denied policy",
		AppliesTo:      "risk_assessment",
		Mode:           "advisory",
	}, member.Token, orgID); status != http.StatusForbidden {
		t.Fatalf("expected org-member policy create to be forbidden, got %d", status)
	}

	memberPolicies := getListAuth[types.Policy](t, server.URL+"/api/v1/policies", member.Token, orgID, http.StatusOK)
	if len(memberPolicies) == 0 {
		t.Fatalf("expected org-member policy reads to be allowed")
	}

	policy := postItemAuth[types.Policy](t, server.URL+"/api/v1/policies", types.CreatePolicyRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Tenant Scoped Policy",
		Code:           "tenant-scoped-policy",
		AppliesTo:      "risk_assessment",
		Mode:           "advisory",
	}, admin.Token, orgID)

	updatedName := "Denied update"
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/policies/"+policy.ID, types.UpdatePolicyRequest{
		Name: &updatedName,
	}, member.Token, orgID); status != http.StatusForbidden {
		t.Fatalf("expected org-member policy update to be forbidden, got %d", status)
	}

	if status := requestStatus(t, http.MethodGet, server.URL+"/api/v1/policies/"+policy.ID, nil, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-tenant policy show to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/policies/"+policy.ID, types.UpdatePolicyRequest{
		Name: &updatedName,
	}, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-tenant policy update to be forbidden, got %d", status)
	}
}

func containsPolicyID(items []types.Policy, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func containsPolicyDecisionCode(items []types.PolicyDecision, code string) bool {
	for _, item := range items {
		if item.PolicyCode == code {
			return true
		}
	}
	return false
}

func decisionByCode(items []types.PolicyDecision, code string) types.PolicyDecision {
	for _, item := range items {
		if item.PolicyCode == code {
			return item
		}
	}
	return types.PolicyDecision{}
}

func hasAuditEventAction(events []types.AuditEvent, action, resourceID string) bool {
	for _, event := range events {
		if event.Action == action && event.ResourceID == resourceID {
			return true
		}
	}
	return false
}

func hasStatusEventType(events []types.StatusEvent, eventType string) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}

func requestStatusAndBody(t *testing.T, method, url string, body any, token, organizationID string) (int, []byte) {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if organizationID != "" {
		req.Header.Set("X-CCP-Organization-ID", organizationID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, data
}

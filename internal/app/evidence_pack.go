package app

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	policylib "github.com/change-control-plane/change-control-plane/internal/policies"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) buildRolloutExecutionDetail(ctx context.Context, runtimeContext types.RolloutExecutionRuntimeContext) (types.RolloutExecutionDetail, error) {
	timeline, err := a.Store.ListAuditEvents(ctx, storage.AuditEventQuery{
		OrganizationID: runtimeContext.Execution.OrganizationID,
		ProjectID:      runtimeContext.Execution.ProjectID,
		ResourceType:   "rollout_execution",
		ResourceID:     runtimeContext.Execution.ID,
		Limit:          100,
	})
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}
	statusTimeline, err := a.Store.ListStatusEvents(ctx, storage.StatusEventQuery{
		OrganizationID:     runtimeContext.Execution.OrganizationID,
		ProjectID:          runtimeContext.Execution.ProjectID,
		RolloutExecutionID: runtimeContext.Execution.ID,
		Limit:              200,
	})
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}

	summary := types.RolloutExecutionRuntimeSummary{
		BackendType:     runtimeContext.Execution.BackendType,
		BackendStatus:   runtimeContext.Execution.BackendStatus,
		ProgressPercent: runtimeContext.Execution.ProgressPercent,
	}
	if runtimeContext.BackendIntegration != nil {
		summary.ControlMode = normalizeIntegrationMode(runtimeContext.BackendIntegration.Mode)
		summary.ControlEnabled = integrationAllowsActiveControl(runtimeContext.BackendIntegration)
		summary.AdvisoryOnly = advisoryOnlyRuntime(runtimeContext)
		if summary.AdvisoryOnly {
			summary.ControlRationale = "Advisory mode is active for the live backend integration, so external submit, pause, resume, and rollback actions are suppressed."
		} else if summary.ControlMode == "active_control" {
			summary.ControlRationale = "Active control is enabled for the backend integration, so provider actions can be executed."
		}
	}
	if runtimeContext.Execution.Metadata != nil {
		summary.RecommendedAction = metadataString(runtimeContext.Execution.Metadata, "recommended_action")
		summary.LastProviderAction = metadataString(runtimeContext.Execution.Metadata, "last_provider_action")
		summary.LastActionDisposition = metadataString(runtimeContext.Execution.Metadata, "last_action_disposition")
		summary.LastProviderActionSummary = metadataString(runtimeContext.Execution.Metadata, "backend_summary")
		if summary.ControlMode == "" {
			summary.ControlMode = metadataString(runtimeContext.Execution.Metadata, "control_mode")
		}
		if summary.ControlRationale == "" {
			summary.ControlRationale = metadataString(runtimeContext.Execution.Metadata, "control_rationale")
		}
		if summary.RecommendedAction != "" {
			summary.AdvisoryOnly = true
		}
	}
	if len(runtimeContext.SignalSnapshots) > 0 {
		latest := runtimeContext.SignalSnapshots[len(runtimeContext.SignalSnapshots)-1]
		summary.LatestSignalHealth = latest.Health
		summary.LatestSignalSummary = latest.Summary
	}
	if len(runtimeContext.VerificationResults) > 0 {
		latest := runtimeContext.VerificationResults[len(runtimeContext.VerificationResults)-1]
		summary.LatestDecision = latest.Decision
		if latest.Automated {
			summary.LatestDecisionMode = "automated"
		} else {
			summary.LatestDecisionMode = valueOrDefault(latest.DecisionSource, "manual")
		}
	}
	verificationResults := make([]types.VerificationResult, 0, len(runtimeContext.VerificationResults))
	for _, item := range runtimeContext.VerificationResults {
		verificationResults = append(verificationResults, decorateVerificationResult(item, summary))
	}

	return types.RolloutExecutionDetail{
		Execution:               runtimeContext.Execution,
		VerificationResults:     verificationResults,
		SignalSnapshots:         runtimeContext.SignalSnapshots,
		Timeline:                timeline,
		StatusTimeline:          statusTimeline,
		EffectiveRollbackPolicy: runtimeContext.EffectiveRollbackPolicy,
		RuntimeSummary:          summary,
	}, nil
}

func (a *Application) GetRolloutEvidencePack(ctx context.Context, id string) (types.RolloutEvidencePack, error) {
	runtimeContext, err := a.GetRolloutExecutionRuntimeContext(ctx, id)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	detail, err := a.buildRolloutExecutionDetail(ctx, runtimeContext)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	organization, err := a.Store.GetOrganization(ctx, runtimeContext.Execution.OrganizationID)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	project, err := a.Store.GetProject(ctx, runtimeContext.Execution.ProjectID)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	policyDecisions, err := a.listEvidencePolicyDecisions(ctx, runtimeContext)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	repositories, err := a.listEvidenceRepositories(ctx, runtimeContext)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	discoveredResources, err := a.listEvidenceDiscoveredResources(ctx, runtimeContext)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	graphRelationships, err := a.listEvidenceGraphRelationships(ctx, runtimeContext, repositories, discoveredResources)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	incidents, err := a.Incidents(ctx, IncidentQuery{
		ProjectID:     runtimeContext.Execution.ProjectID,
		ServiceID:     runtimeContext.Execution.ServiceID,
		EnvironmentID: runtimeContext.Execution.EnvironmentID,
		ChangeSetID:   runtimeContext.Execution.ChangeSetID,
		Limit:         50,
	})
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}
	auditTrail, err := a.listEvidenceAuditTrail(ctx, runtimeContext)
	if err != nil {
		return types.RolloutEvidencePack{}, err
	}

	return types.RolloutEvidencePack{
		Summary:             buildRolloutEvidencePackSummary(detail, runtimeContext, policyDecisions, incidents, repositories, discoveredResources),
		Organization:        organization,
		Project:             project,
		Service:             runtimeContext.Service,
		Environment:         runtimeContext.Environment,
		ChangeSet:           runtimeContext.ChangeSet,
		Assessment:          runtimeContext.Assessment,
		Plan:                runtimeContext.Plan,
		ExecutionDetail:     detail,
		BackendIntegration:  runtimeContext.BackendIntegration,
		SignalIntegration:   runtimeContext.SignalIntegration,
		PolicyDecisions:     policyDecisions,
		Incidents:           incidents,
		Repositories:        repositories,
		DiscoveredResources: discoveredResources,
		GraphRelationships:  graphRelationships,
		AuditTrail:          auditTrail,
	}, nil
}

func (a *Application) listEvidencePolicyDecisions(ctx context.Context, runtimeContext types.RolloutExecutionRuntimeContext) ([]types.PolicyDecision, error) {
	queries := []storage.PolicyDecisionQuery{
		{ProjectID: runtimeContext.Execution.ProjectID, ChangeSetID: runtimeContext.ChangeSet.ID, Limit: 100},
		{ProjectID: runtimeContext.Execution.ProjectID, RiskAssessmentID: runtimeContext.Assessment.ID, Limit: 100},
		{ProjectID: runtimeContext.Execution.ProjectID, RolloutPlanID: runtimeContext.Plan.ID, Limit: 100},
		{ProjectID: runtimeContext.Execution.ProjectID, RolloutExecutionID: runtimeContext.Execution.ID, Limit: 100},
	}
	merged := make([]types.PolicyDecision, 0, 16)
	seen := map[string]struct{}{}
	for _, query := range queries {
		items, err := a.ListPolicyDecisions(ctx, query)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			merged = append(merged, item)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].CreatedAt.Equal(merged[j].CreatedAt) {
			return merged[i].ID < merged[j].ID
		}
		return merged[i].CreatedAt.Before(merged[j].CreatedAt)
	})
	return merged, nil
}

func (a *Application) listEvidenceRepositories(ctx context.Context, runtimeContext types.RolloutExecutionRuntimeContext) ([]types.Repository, error) {
	items, err := a.ListRepositories(ctx, storage.RepositoryQuery{
		ProjectID: runtimeContext.Execution.ProjectID,
		ServiceID: runtimeContext.Execution.ServiceID,
		Limit:     200,
	})
	if err != nil {
		return nil, err
	}
	filtered := make([]types.Repository, 0, len(items))
	for _, item := range items {
		if item.EnvironmentID != "" && item.EnvironmentID != runtimeContext.Execution.EnvironmentID {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Name == filtered[j].Name {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].Name < filtered[j].Name
	})
	return filtered, nil
}

func (a *Application) listEvidenceDiscoveredResources(ctx context.Context, runtimeContext types.RolloutExecutionRuntimeContext) ([]types.DiscoveredResource, error) {
	items, err := a.ListDiscoveredResources(ctx, storage.DiscoveredResourceQuery{
		ProjectID: runtimeContext.Execution.ProjectID,
		ServiceID: runtimeContext.Execution.ServiceID,
		Limit:     500,
	})
	if err != nil {
		return nil, err
	}
	filtered := make([]types.DiscoveredResource, 0, len(items))
	for _, item := range items {
		if item.EnvironmentID != "" && item.EnvironmentID != runtimeContext.Execution.EnvironmentID {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Name == filtered[j].Name {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].Name < filtered[j].Name
	})
	return filtered, nil
}

func (a *Application) listEvidenceGraphRelationships(ctx context.Context, runtimeContext types.RolloutExecutionRuntimeContext, repositories []types.Repository, discoveredResources []types.DiscoveredResource) ([]types.GraphRelationship, error) {
	items, err := a.Store.ListGraphRelationships(ctx, storage.GraphRelationshipQuery{
		OrganizationID: runtimeContext.Execution.OrganizationID,
		Limit:          2000,
	})
	if err != nil {
		return nil, err
	}
	relevantIDs := map[string]struct{}{
		runtimeContext.Execution.ServiceID:     {},
		runtimeContext.Execution.EnvironmentID: {},
		runtimeContext.Execution.ChangeSetID:   {},
		runtimeContext.Execution.ID:            {},
	}
	for _, item := range repositories {
		relevantIDs[item.ID] = struct{}{}
	}
	for _, item := range discoveredResources {
		relevantIDs[item.ID] = struct{}{}
	}

	filtered := make([]types.GraphRelationship, 0, len(items))
	for _, item := range items {
		if _, ok := relevantIDs[item.FromResourceID]; ok {
			filtered = append(filtered, item)
			continue
		}
		if _, ok := relevantIDs[item.ToResourceID]; ok {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].LastObservedAt.Equal(filtered[j].LastObservedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].LastObservedAt.After(filtered[j].LastObservedAt)
	})
	return filtered, nil
}

func (a *Application) listEvidenceAuditTrail(ctx context.Context, runtimeContext types.RolloutExecutionRuntimeContext) ([]types.AuditEvent, error) {
	keys := []struct {
		resourceType string
		resourceID   string
	}{
		{resourceType: "change_set", resourceID: runtimeContext.ChangeSet.ID},
		{resourceType: "risk_assessment", resourceID: runtimeContext.Assessment.ID},
		{resourceType: "rollout_plan", resourceID: runtimeContext.Plan.ID},
		{resourceType: "rollout_execution", resourceID: runtimeContext.Execution.ID},
	}
	merged := make([]types.AuditEvent, 0, 32)
	seen := map[string]struct{}{}
	for _, key := range keys {
		if strings.TrimSpace(key.resourceID) == "" {
			continue
		}
		items, err := a.Store.ListAuditEvents(ctx, storage.AuditEventQuery{
			OrganizationID: runtimeContext.Execution.OrganizationID,
			ProjectID:      runtimeContext.Execution.ProjectID,
			ResourceType:   key.resourceType,
			ResourceID:     key.resourceID,
			Limit:          100,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			merged = append(merged, item)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].CreatedAt.Equal(merged[j].CreatedAt) {
			return merged[i].ID < merged[j].ID
		}
		return merged[i].CreatedAt.Before(merged[j].CreatedAt)
	})
	return merged, nil
}

func buildRolloutEvidencePackSummary(detail types.RolloutExecutionDetail, runtimeContext types.RolloutExecutionRuntimeContext, policyDecisions []types.PolicyDecision, incidents []types.Incident, repositories []types.Repository, discoveredResources []types.DiscoveredResource) types.RolloutEvidencePackSummary {
	blockingPolicies := 0
	manualReviewPolicies := 0
	for _, decision := range policyDecisions {
		switch decision.Outcome {
		case policylib.ModeBlock:
			blockingPolicies++
		case policylib.ModeRequireManualReview:
			manualReviewPolicies++
		}
	}
	latestVerificationOutcome := ""
	if len(detail.VerificationResults) > 0 {
		latestVerificationOutcome = detail.VerificationResults[len(detail.VerificationResults)-1].Outcome
	}
	highlights := []string{
		"Risk review scored this rollout " + string(runtimeContext.Assessment.Level) + " at " + intToString(runtimeContext.Assessment.Score) + ".",
	}
	if strings.TrimSpace(runtimeContext.Assessment.BlastRadius.Summary) != "" {
		highlights = append(highlights, runtimeContext.Assessment.BlastRadius.Summary)
	}
	if runtimeContext.Plan.ApprovalRequired {
		highlights = append(highlights, "Approval level "+valueOrDefault(runtimeContext.Plan.ApprovalLevel, "required")+" is in effect for this rollout.")
	}
	if blockingPolicies > 0 || manualReviewPolicies > 0 {
		highlights = append(highlights, "Policy decisions include "+intToString(blockingPolicies)+" blocking and "+intToString(manualReviewPolicies)+" manual-review outcomes.")
	}
	highlights = append(highlights, "Mapped evidence currently spans "+intToString(len(repositories))+" repositories, "+intToString(len(discoveredResources))+" runtime resources, and "+intToString(len(incidents))+" related incidents.")
	if latestVerificationOutcome != "" || detail.RuntimeSummary.LatestDecision != "" {
		highlights = append(highlights, "Latest verification outcome is "+valueOrDefault(latestVerificationOutcome, "unrecorded")+" with decision "+valueOrDefault(detail.RuntimeSummary.LatestDecision, "pending")+".")
	}
	return types.RolloutEvidencePackSummary{
		GeneratedAt:               time.Now().UTC(),
		ApprovalState:             rolloutEvidenceApprovalState(runtimeContext.Plan, runtimeContext.Execution),
		RiskLevel:                 runtimeContext.Assessment.Level,
		RiskScore:                 runtimeContext.Assessment.Score,
		BlastRadiusScope:          runtimeContext.Assessment.BlastRadius.Scope,
		BlastRadiusSummary:        runtimeContext.Assessment.BlastRadius.Summary,
		RolloutStrategy:           runtimeContext.Plan.Strategy,
		LatestDecision:            valueOrDefault(detail.RuntimeSummary.LatestDecision, runtimeContext.Execution.LastDecision),
		LatestVerificationOutcome: latestVerificationOutcome,
		ControlMode:               detail.RuntimeSummary.ControlMode,
		IncidentCount:             len(incidents),
		RepositoryCount:           len(repositories),
		DiscoveredResourceCount:   len(discoveredResources),
		BlockingPolicyCount:       blockingPolicies,
		ManualReviewPolicyCount:   manualReviewPolicies,
		EvidenceHighlights:        highlights,
	}
}

func rolloutEvidenceApprovalState(plan types.RolloutPlan, execution types.RolloutExecution) string {
	if !plan.ApprovalRequired {
		return "not_required"
	}
	switch execution.Status {
	case "awaiting_approval":
		return "pending"
	case "approved", "in_progress", "paused", "verified", "completed", "rolled_back", "failed":
		return "satisfied"
	default:
		return "required"
	}
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

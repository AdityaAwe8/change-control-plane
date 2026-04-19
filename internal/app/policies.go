package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/common"
	policylib "github.com/change-control-plane/change-control-plane/internal/policies"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) CreatePolicy(ctx context.Context, req types.CreatePolicyRequest) (types.Policy, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Policy{}, err
	}

	orgID := strings.TrimSpace(req.OrganizationID)
	if orgID == "" {
		return types.Policy{}, fmt.Errorf("%w: organization_id is required", ErrValidation)
	}
	if !a.Authorizer.CanManagePolicies(identity, orgID, strings.TrimSpace(req.ProjectID)) {
		return types.Policy{}, a.forbidden(ctx, identity, "policy.create.denied", "policy", "", orgID, strings.TrimSpace(req.ProjectID), []string{"actor lacks policy management permission"})
	}

	now := time.Now().UTC()
	policy := types.Policy{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("pol"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID: orgID,
		ProjectID:      strings.TrimSpace(req.ProjectID),
		ServiceID:      strings.TrimSpace(req.ServiceID),
		EnvironmentID:  strings.TrimSpace(req.EnvironmentID),
		Name:           strings.TrimSpace(req.Name),
		Code:           normalizePolicyCode(req.Code, req.Name),
		AppliesTo:      policylib.NormalizeAppliesTo(req.AppliesTo),
		Mode:           policylib.NormalizeMode(req.Mode),
		Enabled:        req.Enabled == nil || *req.Enabled,
		Priority:       req.Priority,
		Description:    strings.TrimSpace(req.Description),
		Conditions:     policylib.NormalizeConditions(req.Conditions),
	}
	if err := a.preparePolicy(ctx, &policy); err != nil {
		return types.Policy{}, err
	}
	if err := a.ensurePolicyCodeUnique(ctx, policy.OrganizationID, "", policy.Code); err != nil {
		return types.Policy{}, err
	}
	if err := a.Store.CreatePolicy(ctx, policy); err != nil {
		return types.Policy{}, err
	}
	if err := a.record(ctx, identity, "policy.created", "policy", policy.ID, policy.OrganizationID, policy.ProjectID, []string{policy.Name, policy.Mode, policy.AppliesTo}, withStatusCategory("governance"), withStatusSummary("policy created")); err != nil {
		return types.Policy{}, err
	}
	return policy, nil
}

func (a *Application) ListPolicies(ctx context.Context) ([]types.Policy, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanViewPolicies(identity, orgID, "") {
		return nil, ErrForbidden
	}
	return a.Store.ListPolicies(ctx, storage.PolicyQuery{OrganizationID: orgID})
}

func (a *Application) GetPolicy(ctx context.Context, id string) (types.Policy, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Policy{}, err
	}
	policy, err := a.Store.GetPolicy(ctx, id)
	if err != nil {
		return types.Policy{}, err
	}
	if !a.Authorizer.CanViewPolicies(identity, policy.OrganizationID, policy.ProjectID) {
		return types.Policy{}, ErrForbidden
	}
	return policy, nil
}

func (a *Application) UpdatePolicy(ctx context.Context, id string, req types.UpdatePolicyRequest) (types.Policy, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Policy{}, err
	}
	policy, err := a.Store.GetPolicy(ctx, id)
	if err != nil {
		return types.Policy{}, err
	}
	if !a.Authorizer.CanManagePolicies(identity, policy.OrganizationID, policy.ProjectID) {
		return types.Policy{}, a.forbidden(ctx, identity, "policy.update.denied", "policy", policy.ID, policy.OrganizationID, policy.ProjectID, []string{"actor lacks policy management permission"})
	}

	if req.ProjectID != nil {
		policy.ProjectID = strings.TrimSpace(*req.ProjectID)
	}
	if req.ServiceID != nil {
		policy.ServiceID = strings.TrimSpace(*req.ServiceID)
	}
	if req.EnvironmentID != nil {
		policy.EnvironmentID = strings.TrimSpace(*req.EnvironmentID)
	}
	if req.Name != nil {
		policy.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		policy.Code = normalizePolicyCode(*req.Code, policy.Name)
	}
	if req.AppliesTo != nil {
		policy.AppliesTo = policylib.NormalizeAppliesTo(*req.AppliesTo)
	}
	if req.Mode != nil {
		policy.Mode = policylib.NormalizeMode(*req.Mode)
	}
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		policy.Priority = *req.Priority
	}
	if req.Description != nil {
		policy.Description = strings.TrimSpace(*req.Description)
	}
	if req.Conditions != nil {
		policy.Conditions = policylib.NormalizeConditions(*req.Conditions)
	}
	if req.Metadata != nil {
		policy.Metadata = req.Metadata
	}
	policy.UpdatedAt = time.Now().UTC()
	if err := a.preparePolicy(ctx, &policy); err != nil {
		return types.Policy{}, err
	}
	if err := a.ensurePolicyCodeUnique(ctx, policy.OrganizationID, policy.ID, policy.Code); err != nil {
		return types.Policy{}, err
	}
	if err := a.Store.UpdatePolicy(ctx, policy); err != nil {
		return types.Policy{}, err
	}
	if err := a.record(ctx, identity, "policy.updated", "policy", policy.ID, policy.OrganizationID, policy.ProjectID, []string{policy.Name, policy.Mode, policy.AppliesTo}, withStatusCategory("governance"), withStatusSummary("policy updated")); err != nil {
		return types.Policy{}, err
	}
	return policy, nil
}

func (a *Application) ListPolicyDecisions(ctx context.Context, query storage.PolicyDecisionQuery) ([]types.PolicyDecision, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanViewPolicies(identity, orgID, query.ProjectID) {
		return nil, ErrForbidden
	}
	query.OrganizationID = orgID
	return a.Store.ListPolicyDecisions(ctx, query)
}

func (a *Application) seedDefaultPolicies(ctx context.Context, organizationID string) error {
	existing, err := a.Store.ListPolicies(ctx, storage.PolicyQuery{OrganizationID: organizationID, Limit: 1})
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	now := time.Now().UTC()
	for _, policy := range defaultPoliciesForOrganization(organizationID, now) {
		if err := a.Store.CreatePolicy(ctx, policy); err != nil {
			return err
		}
	}
	return nil
}

type policyEvaluationReference struct {
	riskAssessmentID   string
	rolloutPlanID      string
	rolloutExecutionID string
	metadata           types.Metadata
}

func (a *Application) evaluateAndPersistPolicies(ctx context.Context, appliesTo string, change types.ChangeSet, service types.Service, environment types.Environment, assessment types.RiskAssessment, reference policyEvaluationReference) ([]types.PolicyDecision, error) {
	decisions, err := a.evaluatePolicies(ctx, appliesTo, change, service, environment, assessment, reference)
	if err != nil {
		return nil, err
	}
	if err := a.persistPolicyDecisions(ctx, decisions); err != nil {
		return nil, err
	}
	return decisions, nil
}

func (a *Application) evaluatePolicies(ctx context.Context, appliesTo string, change types.ChangeSet, service types.Service, environment types.Environment, assessment types.RiskAssessment, reference policyEvaluationReference) ([]types.PolicyDecision, error) {
	policies, err := a.resolvePoliciesForWorkflow(ctx, change.OrganizationID, change.ProjectID, change.ServiceID, change.EnvironmentID, appliesTo)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	decisions := make([]types.PolicyDecision, 0, len(policies))
	for _, policy := range policies {
		reasons, matched := policylib.EvaluatePolicy(policy, policylib.EvaluationInput{
			Change:      change,
			Service:     service,
			Environment: environment,
			Assessment:  assessment,
			AppliesTo:   appliesTo,
		})
		if !matched {
			continue
		}
		decision := types.PolicyDecision{
			BaseRecord: types.BaseRecord{
				ID:        common.NewID("pdec"),
				CreatedAt: now,
				UpdatedAt: now,
				Metadata:  cloneMetadata(reference.metadata),
			},
			OrganizationID:    change.OrganizationID,
			ProjectID:         change.ProjectID,
			ServiceID:         change.ServiceID,
			EnvironmentID:     change.EnvironmentID,
			PolicyID:          policy.ID,
			PolicyName:        policy.Name,
			PolicyCode:        policy.Code,
			PolicyScope:       policy.Scope,
			AppliesTo:         appliesTo,
			Mode:              policy.Mode,
			ChangeSetID:       change.ID,
			RiskAssessmentID:  reference.riskAssessmentID,
			RolloutPlanID:     reference.rolloutPlanID,
			RolloutExecutionID: reference.rolloutExecutionID,
			Outcome:           policy.Mode,
			Summary:           policyDecisionSummary(policy, appliesTo, reasons),
			Reasons:           reasons,
		}
		if decision.Metadata == nil {
			decision.Metadata = types.Metadata{}
		}
		decision.Metadata["policy_scope"] = policy.Scope
		decision.Metadata["matched_triggers"] = append([]string(nil), policy.Triggers...)
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

func (a *Application) persistPolicyDecisions(ctx context.Context, decisions []types.PolicyDecision) error {
	for _, decision := range decisions {
		if err := a.Store.CreatePolicyDecision(ctx, decision); err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) resolvePoliciesForWorkflow(ctx context.Context, organizationID, projectID, serviceID, environmentID, appliesTo string) ([]types.Policy, error) {
	items, err := a.Store.ListPolicies(ctx, storage.PolicyQuery{
		OrganizationID: organizationID,
		AppliesTo:      policylib.NormalizeAppliesTo(appliesTo),
		EnabledOnly:    true,
	})
	if err != nil {
		return nil, err
	}
	candidates := make([]types.Policy, 0, len(items))
	for _, item := range items {
		if item.ProjectID != "" && item.ProjectID != projectID {
			continue
		}
		if item.ServiceID != "" && item.ServiceID != serviceID {
			continue
		}
		if item.EnvironmentID != "" && item.EnvironmentID != environmentID {
			continue
		}
		candidates = append(candidates, item)
	}
	slices.SortFunc(candidates, comparePolicies)
	return candidates, nil
}

func comparePolicies(left, right types.Policy) int {
	leftSpecificity := policySpecificity(left)
	rightSpecificity := policySpecificity(right)
	if leftSpecificity != rightSpecificity {
		if leftSpecificity > rightSpecificity {
			return -1
		}
		return 1
	}
	if left.Priority != right.Priority {
		if left.Priority > right.Priority {
			return -1
		}
		return 1
	}
	if left.CreatedAt.After(right.CreatedAt) {
		return -1
	}
	if left.CreatedAt.Before(right.CreatedAt) {
		return 1
	}
	return 0
}

func policySpecificity(policy types.Policy) int {
	score := 0
	if policy.ProjectID != "" {
		score++
	}
	if policy.ServiceID != "" {
		score++
	}
	if policy.EnvironmentID != "" {
		score++
	}
	return score
}

func policyDecisionSummary(policy types.Policy, appliesTo string, reasons []string) string {
	switch policy.Mode {
	case policylib.ModeBlock:
		return fmt.Sprintf("%s blocks %s when %s.", policy.Name, strings.ReplaceAll(appliesTo, "_", " "), strings.Join(reasons, "; "))
	case policylib.ModeRequireManualReview:
		return fmt.Sprintf("%s requires manual review when %s.", policy.Name, strings.Join(reasons, "; "))
	default:
		return fmt.Sprintf("%s advises caution because %s.", policy.Name, strings.Join(reasons, "; "))
	}
}

func anyPolicyDecision(decisions []types.PolicyDecision, outcome string) bool {
	for _, decision := range decisions {
		if decision.Outcome == outcome {
			return true
		}
	}
	return false
}

func blockingPolicyNames(decisions []types.PolicyDecision) []string {
	names := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		if decision.Outcome == policylib.ModeBlock {
			names = append(names, decision.PolicyName)
		}
	}
	return names
}

func decisionSummaries(decisions []types.PolicyDecision) []string {
	items := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		items = append(items, decision.Summary)
	}
	return items
}

func decisionScope(decisions []types.PolicyDecision) statusScope {
	scope := statusScope{}
	for _, decision := range decisions {
		scope.projectID = valueOrDefault(scope.projectID, decision.ProjectID)
		scope.serviceID = valueOrDefault(scope.serviceID, decision.ServiceID)
		scope.environmentID = valueOrDefault(scope.environmentID, decision.EnvironmentID)
		scope.changeSetID = valueOrDefault(scope.changeSetID, decision.ChangeSetID)
	}
	return scope
}

func (a *Application) recordPolicyBlockedOutcome(ctx context.Context, identity auth.Identity, change types.ChangeSet, decisions []types.PolicyDecision) error {
	details := append([]string{"rollout plan blocked by policy"}, blockingPolicyNames(decisions)...)
	return a.record(
		ctx,
		identity,
		"policy.blocked",
		"change_set",
		change.ID,
		change.OrganizationID,
		change.ProjectID,
		details,
		withStatusCategory("governance"),
		withStatusSeverity("warning"),
		withStatusSummary("rollout plan blocked by policy"),
		withStatusScope(decisionScope(decisions)),
	)
}

func (a *Application) preparePolicy(ctx context.Context, policy *types.Policy) error {
	policy.Name = strings.TrimSpace(policy.Name)
	policy.Code = normalizePolicyCode(policy.Code, policy.Name)
	policy.AppliesTo = policylib.NormalizeAppliesTo(policy.AppliesTo)
	policy.Mode = policylib.NormalizeMode(policy.Mode)
	policy.Scope = policylib.DetermineScope(policy.ProjectID, policy.ServiceID, policy.EnvironmentID)
	policy.Conditions = policylib.NormalizeConditions(policy.Conditions)
	policy.Triggers = policylib.ComputeTriggers(policy.Conditions)

	if policy.Name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if policy.Code == "" {
		return fmt.Errorf("%w: code is required", ErrValidation)
	}
	if err := policylib.ValidateAppliesTo(policy.AppliesTo); err != nil {
		return fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	if err := policylib.ValidateMode(policy.Mode); err != nil {
		return fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	if err := policylib.ValidateConditions(policy.Conditions); err != nil {
		return fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	if err := a.validatePolicyScope(ctx, policy.OrganizationID, policy.ProjectID, policy.ServiceID, policy.EnvironmentID); err != nil {
		return err
	}
	return nil
}

func (a *Application) validatePolicyScope(ctx context.Context, organizationID, projectID, serviceID, environmentID string) error {
	if strings.TrimSpace(projectID) != "" {
		project, err := a.Store.GetProject(ctx, projectID)
		if err != nil {
			return fmt.Errorf("%w: project %s", storage.ErrNotFound, projectID)
		}
		if project.OrganizationID != organizationID {
			return fmt.Errorf("%w: policy project scope does not belong to organization", ErrValidation)
		}
	}
	if strings.TrimSpace(serviceID) != "" {
		service, err := a.Store.GetService(ctx, serviceID)
		if err != nil {
			return fmt.Errorf("%w: service %s", storage.ErrNotFound, serviceID)
		}
		if service.OrganizationID != organizationID {
			return fmt.Errorf("%w: policy service scope does not belong to organization", ErrValidation)
		}
		if projectID != "" && service.ProjectID != projectID {
			return fmt.Errorf("%w: policy service scope does not match project", ErrValidation)
		}
	}
	if strings.TrimSpace(environmentID) != "" {
		environment, err := a.Store.GetEnvironment(ctx, environmentID)
		if err != nil {
			return fmt.Errorf("%w: environment %s", storage.ErrNotFound, environmentID)
		}
		if environment.OrganizationID != organizationID {
			return fmt.Errorf("%w: policy environment scope does not belong to organization", ErrValidation)
		}
		if projectID != "" && environment.ProjectID != projectID {
			return fmt.Errorf("%w: policy environment scope does not match project", ErrValidation)
		}
	}
	return nil
}

func (a *Application) ensurePolicyCodeUnique(ctx context.Context, organizationID, currentPolicyID, code string) error {
	items, err := a.Store.ListPolicies(ctx, storage.PolicyQuery{OrganizationID: organizationID})
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == currentPolicyID {
			continue
		}
		if strings.EqualFold(item.Code, code) {
			return fmt.Errorf("%w: policy code %s already exists", ErrValidation, code)
		}
	}
	return nil
}

func normalizePolicyCode(code, name string) string {
	source := strings.TrimSpace(code)
	if source == "" {
		source = strings.TrimSpace(name)
	}
	source = strings.ToLower(source)
	var builder strings.Builder
	lastDash := false
	for _, r := range source {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if builder.Len() == 0 || lastDash {
				continue
			}
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return ""
	}
	return result
}

func defaultPoliciesForOrganization(organizationID string, now time.Time) []types.Policy {
	items := []types.Policy{
		{
			BaseRecord: types.BaseRecord{
				ID:        common.NewID("pol"),
				CreatedAt: now,
				UpdatedAt: now,
			},
			OrganizationID: organizationID,
			Name:           "Production High Risk Approval",
			Code:           "prod-high-risk-approval",
			AppliesTo:      policylib.AppliesToRolloutPlan,
			Mode:           policylib.ModeRequireManualReview,
			Enabled:        true,
			Priority:       100,
			Description:    "Require manual review before planning production rollouts when the deterministic risk assessment is high or critical.",
			Conditions: types.PolicyCondition{
				ProductionOnly: true,
				MinRiskLevel:   string(types.RiskLevelHigh),
			},
		},
		{
			BaseRecord: types.BaseRecord{
				ID:        common.NewID("pol"),
				CreatedAt: now.Add(1 * time.Millisecond),
				UpdatedAt: now.Add(1 * time.Millisecond),
			},
			OrganizationID: organizationID,
			Name:           "Observability Coverage Advisory",
			Code:           "observability-coverage-advisory",
			AppliesTo:      policylib.AppliesToRiskAssessment,
			Mode:           policylib.ModeAdvisory,
			Enabled:        true,
			Priority:       50,
			Description:    "Warn when rollout confidence is reduced by missing observability or SLO coverage.",
			Conditions: types.PolicyCondition{
				MissingCapabilities: []string{"observability", "slo"},
			},
		},
		{
			BaseRecord: types.BaseRecord{
				ID:        common.NewID("pol"),
				CreatedAt: now.Add(2 * time.Millisecond),
				UpdatedAt: now.Add(2 * time.Millisecond),
			},
			OrganizationID: organizationID,
			Name:           "Critical Schema Change Freeze",
			Code:           "critical-schema-change-freeze",
			AppliesTo:      policylib.AppliesToRolloutPlan,
			Mode:           policylib.ModeBlock,
			Enabled:        true,
			Priority:       90,
			Description:    "Block rollout planning when a production schema change also reaches critical risk.",
			Conditions: types.PolicyCondition{
				ProductionOnly:  true,
				MinRiskLevel:    string(types.RiskLevelCritical),
				RequiredTouches: []string{"schema"},
			},
		},
	}
	for index := range items {
		items[index].Conditions = policylib.NormalizeConditions(items[index].Conditions)
		items[index].Scope = policylib.DetermineScope(items[index].ProjectID, items[index].ServiceID, items[index].EnvironmentID)
		items[index].Triggers = policylib.ComputeTriggers(items[index].Conditions)
	}
	return items
}

func cloneMetadata(metadata types.Metadata) types.Metadata {
	if metadata == nil {
		return nil
	}
	cloned := make(types.Metadata, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func policyDecisionMetadata(outcome string, blocked bool) types.Metadata {
	metadata := types.Metadata{
		"evaluation_outcome": outcome,
	}
	if blocked {
		metadata["blocked_attempt"] = true
	}
	return metadata
}

func isPolicyBlocked(decisions []types.PolicyDecision) bool {
	return anyPolicyDecision(decisions, policylib.ModeBlock)
}

func isPolicyReviewRequired(decisions []types.PolicyDecision) bool {
	return anyPolicyDecision(decisions, policylib.ModeRequireManualReview)
}

func wrapPolicyBlockError(decisions []types.PolicyDecision) error {
	names := blockingPolicyNames(decisions)
	if len(names) == 0 {
		names = append(names, "matching policy")
	}
	return fmt.Errorf("%w: rollout plan blocked by policy: %s", ErrValidation, strings.Join(names, ", "))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func findPolicyByCode(items []types.Policy, code string) (types.Policy, bool) {
	for _, item := range items {
		if item.Code == code {
			return item, true
		}
	}
	return types.Policy{}, false
}

func ensurePolicyFound(err error, id string) error {
	if errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("%w: policy %s", storage.ErrNotFound, id)
	}
	return err
}

package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) CreateRollbackPolicy(ctx context.Context, req types.CreateRollbackPolicyRequest) (types.RollbackPolicy, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RollbackPolicy{}, err
	}
	orgID := strings.TrimSpace(req.OrganizationID)
	if orgID == "" {
		return types.RollbackPolicy{}, fmt.Errorf("%w: organization_id is required", ErrValidation)
	}
	if !a.Authorizer.CanManageRollbackPolicies(identity, orgID, strings.TrimSpace(req.ProjectID)) {
		return types.RollbackPolicy{}, a.forbidden(ctx, identity, "rollback_policy.create.denied", "rollback_policy", "", orgID, strings.TrimSpace(req.ProjectID), []string{"actor lacks rollback policy permission"})
	}
	if err := a.validateRollbackPolicyScope(ctx, orgID, req.ProjectID, req.ServiceID, req.EnvironmentID); err != nil {
		return types.RollbackPolicy{}, err
	}

	now := time.Now().UTC()
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rollbackOnProviderFailure := true
	if req.RollbackOnProviderFailure != nil {
		rollbackOnProviderFailure = *req.RollbackOnProviderFailure
	}
	rollbackOnCriticalSignals := true
	if req.RollbackOnCriticalSignals != nil {
		rollbackOnCriticalSignals = *req.RollbackOnCriticalSignals
	}
	policy := types.RollbackPolicy{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("rpol"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:            orgID,
		ProjectID:                 strings.TrimSpace(req.ProjectID),
		ServiceID:                 strings.TrimSpace(req.ServiceID),
		EnvironmentID:             strings.TrimSpace(req.EnvironmentID),
		Name:                      strings.TrimSpace(req.Name),
		Description:               strings.TrimSpace(req.Description),
		Enabled:                   enabled,
		Priority:                  req.Priority,
		MaxErrorRate:              req.MaxErrorRate,
		MaxLatencyMs:              req.MaxLatencyMs,
		MinimumThroughput:         req.MinimumThroughput,
		MaxUnhealthyInstances:     req.MaxUnhealthyInstances,
		MaxRestartRate:            req.MaxRestartRate,
		MaxVerificationFailures:   req.MaxVerificationFailures,
		RollbackOnProviderFailure: rollbackOnProviderFailure,
		RollbackOnCriticalSignals: rollbackOnCriticalSignals,
	}
	if req.MaxUnhealthyInstances == 0 {
		policy.MaxUnhealthyInstances = -1
	}
	if policy.Name == "" {
		return types.RollbackPolicy{}, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if err := a.Store.CreateRollbackPolicy(ctx, policy); err != nil {
		return types.RollbackPolicy{}, err
	}
	if err := a.record(ctx, identity, "rollback_policy.created", "rollback_policy", policy.ID, policy.OrganizationID, policy.ProjectID, []string{policy.Name}, withStatusCategory("governance"), withStatusSummary("rollback policy created")); err != nil {
		return types.RollbackPolicy{}, err
	}
	return policy, nil
}

func (a *Application) ListRollbackPolicies(ctx context.Context) ([]types.RollbackPolicy, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListRollbackPolicies(ctx, storage.RollbackPolicyQuery{OrganizationID: orgID})
}

func (a *Application) UpdateRollbackPolicy(ctx context.Context, id string, req types.UpdateRollbackPolicyRequest) (types.RollbackPolicy, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RollbackPolicy{}, err
	}
	policy, err := a.Store.GetRollbackPolicy(ctx, id)
	if err != nil {
		return types.RollbackPolicy{}, err
	}
	if !a.Authorizer.CanManageRollbackPolicies(identity, policy.OrganizationID, policy.ProjectID) {
		return types.RollbackPolicy{}, a.forbidden(ctx, identity, "rollback_policy.update.denied", "rollback_policy", policy.ID, policy.OrganizationID, policy.ProjectID, []string{"actor lacks rollback policy permission"})
	}
	if req.Name != nil {
		policy.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		policy.Description = strings.TrimSpace(*req.Description)
	}
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		policy.Priority = *req.Priority
	}
	if req.MaxErrorRate != nil {
		policy.MaxErrorRate = *req.MaxErrorRate
	}
	if req.MaxLatencyMs != nil {
		policy.MaxLatencyMs = *req.MaxLatencyMs
	}
	if req.MinimumThroughput != nil {
		policy.MinimumThroughput = *req.MinimumThroughput
	}
	if req.MaxUnhealthyInstances != nil {
		policy.MaxUnhealthyInstances = *req.MaxUnhealthyInstances
	}
	if req.MaxRestartRate != nil {
		policy.MaxRestartRate = *req.MaxRestartRate
	}
	if req.MaxVerificationFailures != nil {
		policy.MaxVerificationFailures = *req.MaxVerificationFailures
	}
	if req.RollbackOnProviderFailure != nil {
		policy.RollbackOnProviderFailure = *req.RollbackOnProviderFailure
	}
	if req.RollbackOnCriticalSignals != nil {
		policy.RollbackOnCriticalSignals = *req.RollbackOnCriticalSignals
	}
	if req.Metadata != nil {
		policy.Metadata = req.Metadata
	}
	policy.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateRollbackPolicy(ctx, policy); err != nil {
		return types.RollbackPolicy{}, err
	}
	if err := a.record(ctx, identity, "rollback_policy.updated", "rollback_policy", policy.ID, policy.OrganizationID, policy.ProjectID, []string{policy.Name}, withStatusCategory("governance"), withStatusSummary("rollback policy updated")); err != nil {
		return types.RollbackPolicy{}, err
	}
	return policy, nil
}

func (a *Application) ListStatusEvents(ctx context.Context, query storage.StatusEventQuery) ([]types.StatusEvent, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	query.OrganizationID = orgID
	return a.Store.ListStatusEvents(ctx, query)
}

func (a *Application) GetStatusEvent(ctx context.Context, id string) (types.StatusEvent, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.StatusEvent{}, err
	}
	event, err := a.Store.GetStatusEvent(ctx, id)
	if err != nil {
		return types.StatusEvent{}, err
	}
	if !a.Authorizer.CanViewStatusHistory(identity, event.OrganizationID, event.ProjectID) {
		return types.StatusEvent{}, ErrForbidden
	}
	return event, nil
}

func (a *Application) ListRolloutExecutionStatusEvents(ctx context.Context, executionID string) ([]types.StatusEvent, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	execution, err := a.Store.GetRolloutExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanViewStatusHistory(identity, execution.OrganizationID, execution.ProjectID) {
		return nil, ErrForbidden
	}
	return a.Store.ListStatusEvents(ctx, storage.StatusEventQuery{
		OrganizationID:     execution.OrganizationID,
		ProjectID:          execution.ProjectID,
		RolloutExecutionID: execution.ID,
	})
}

func (a *Application) resolveEffectiveRollbackPolicy(ctx context.Context, runtime types.RolloutExecutionRuntimeContext) (*types.RollbackPolicy, error) {
	policies, err := a.Store.ListRollbackPolicies(ctx, storage.RollbackPolicyQuery{
		OrganizationID: runtime.Execution.OrganizationID,
		EnabledOnly:    true,
	})
	if err != nil {
		return nil, err
	}
	candidates := make([]types.RollbackPolicy, 0, len(policies))
	for _, policy := range policies {
		if policy.ProjectID != "" && policy.ProjectID != runtime.Execution.ProjectID {
			continue
		}
		if policy.ServiceID != "" && policy.ServiceID != runtime.Execution.ServiceID {
			continue
		}
		if policy.EnvironmentID != "" && policy.EnvironmentID != runtime.Execution.EnvironmentID {
			continue
		}
		candidates = append(candidates, policy)
	}
	if len(candidates) == 0 {
		policy := a.defaultRollbackPolicy(runtime)
		return &policy, nil
	}
	slices.SortFunc(candidates, func(left, right types.RollbackPolicy) int {
		leftScore := rollbackPolicySpecificity(left)
		rightScore := rollbackPolicySpecificity(right)
		if leftScore != rightScore {
			if leftScore > rightScore {
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
	})
	chosen := candidates[0]
	return &chosen, nil
}

func rollbackPolicySpecificity(policy types.RollbackPolicy) int {
	score := 0
	if policy.ProjectID != "" {
		score++
	}
	if policy.ServiceID != "" {
		score += 2
	}
	if policy.EnvironmentID != "" {
		score += 2
	}
	return score
}

func (a *Application) defaultRollbackPolicy(runtime types.RolloutExecutionRuntimeContext) types.RollbackPolicy {
	rollbackOnCriticalSignals := runtime.Environment.Production
	if strings.EqualFold(runtime.Service.Criticality, "mission_critical") || runtime.Service.CustomerFacing {
		rollbackOnCriticalSignals = true
	}
	if runtime.Assessment.Level == types.RiskLevelHigh || runtime.Assessment.Level == types.RiskLevelCritical {
		rollbackOnCriticalSignals = true
	}

	maxErrorRate := 5.0
	maxLatencyMs := 1000.0
	maxRestartRate := 3.0
	maxUnhealthyInstances := 1
	if runtime.Environment.Production || rollbackOnCriticalSignals {
		maxErrorRate = 1.0
		maxLatencyMs = 500.0
		maxRestartRate = 1.0
		maxUnhealthyInstances = 0
	}

	return types.RollbackPolicy{
		BaseRecord: types.BaseRecord{
			ID: "rollback_policy_default",
			Metadata: types.Metadata{
				"source": "built_in_default",
			},
		},
		OrganizationID:             runtime.Execution.OrganizationID,
		ProjectID:                  runtime.Execution.ProjectID,
		ServiceID:                  runtime.Execution.ServiceID,
		EnvironmentID:              runtime.Execution.EnvironmentID,
		Name:                       "Built-in default rollback policy",
		Description:                "Fallback policy used when no persisted override matches the rollout scope.",
		Enabled:                    true,
		Priority:                   -1,
		MaxErrorRate:               maxErrorRate,
		MaxLatencyMs:               maxLatencyMs,
		MaxUnhealthyInstances:      maxUnhealthyInstances,
		MaxRestartRate:             maxRestartRate,
		MaxVerificationFailures:    1,
		RollbackOnProviderFailure:  true,
		RollbackOnCriticalSignals:  rollbackOnCriticalSignals,
	}
}

func (a *Application) validateRollbackPolicyScope(ctx context.Context, organizationID, projectID, serviceID, environmentID string) error {
	if strings.TrimSpace(projectID) != "" {
		project, err := a.Store.GetProject(ctx, strings.TrimSpace(projectID))
		if err != nil {
			return err
		}
		if project.OrganizationID != organizationID {
			return fmt.Errorf("%w: rollback policy project scope does not belong to organization", ErrValidation)
		}
	}
	if strings.TrimSpace(serviceID) != "" {
		service, err := a.Store.GetService(ctx, strings.TrimSpace(serviceID))
		if err != nil {
			return err
		}
		if service.OrganizationID != organizationID {
			return fmt.Errorf("%w: rollback policy service scope does not belong to organization", ErrValidation)
		}
		if projectID != "" && service.ProjectID != projectID {
			return fmt.Errorf("%w: rollback policy service scope does not match project", ErrValidation)
		}
	}
	if strings.TrimSpace(environmentID) != "" {
		environment, err := a.Store.GetEnvironment(ctx, strings.TrimSpace(environmentID))
		if err != nil {
			return err
		}
		if environment.OrganizationID != organizationID {
			return fmt.Errorf("%w: rollback policy environment scope does not belong to organization", ErrValidation)
		}
		if projectID != "" && environment.ProjectID != projectID {
			return fmt.Errorf("%w: rollback policy environment scope does not match project", ErrValidation)
		}
	}
	return nil
}

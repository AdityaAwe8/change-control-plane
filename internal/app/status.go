package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/status"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type statusScope struct {
	projectID          string
	teamID             string
	serviceID          string
	environmentID      string
	rolloutExecutionID string
	changeSetID        string
}

type statusRecordOptions struct {
	category      string
	severity      string
	previousState string
	newState      string
	source        string
	summary       string
	automated     *bool
	correlationID string
	metadata      types.Metadata
	scope         statusScope
}

type statusRecordOption func(*statusRecordOptions)

func withStatusCategory(category string) statusRecordOption {
	return func(options *statusRecordOptions) {
		options.category = strings.TrimSpace(category)
	}
}

func withStatusSeverity(severity string) statusRecordOption {
	return func(options *statusRecordOptions) {
		options.severity = strings.TrimSpace(severity)
	}
}

func withStatusStates(previous, next string) statusRecordOption {
	return func(options *statusRecordOptions) {
		options.previousState = strings.TrimSpace(previous)
		options.newState = strings.TrimSpace(next)
	}
}

func withStatusSource(source string) statusRecordOption {
	return func(options *statusRecordOptions) {
		options.source = strings.TrimSpace(source)
	}
}

func withStatusSummary(summary string) statusRecordOption {
	return func(options *statusRecordOptions) {
		options.summary = strings.TrimSpace(summary)
	}
}

func withStatusAutomated(automated bool) statusRecordOption {
	return func(options *statusRecordOptions) {
		options.automated = &automated
	}
}

func withStatusMetadata(metadata types.Metadata) statusRecordOption {
	return func(options *statusRecordOptions) {
		options.metadata = metadata
	}
}

func withStatusScope(scope statusScope) statusRecordOption {
	return func(options *statusRecordOptions) {
		options.scope = scope
	}
}

func (a *Application) emitStatusEventFromAudit(ctx context.Context, identity auth.Identity, auditEvent types.AuditEvent, details []string, options ...statusRecordOption) error {
	merged := statusRecordOptions{}
	for _, option := range options {
		if option != nil {
			option(&merged)
		}
	}

	scope, err := a.resolveStatusScope(ctx, auditEvent.ResourceType, auditEvent.ResourceID, auditEvent.OrganizationID, auditEvent.ProjectID)
	if err != nil {
		scope = statusScope{projectID: auditEvent.ProjectID}
	}
	if merged.scope.projectID != "" {
		scope.projectID = merged.scope.projectID
	}
	if merged.scope.teamID != "" {
		scope.teamID = merged.scope.teamID
	}
	if merged.scope.serviceID != "" {
		scope.serviceID = merged.scope.serviceID
	}
	if merged.scope.environmentID != "" {
		scope.environmentID = merged.scope.environmentID
	}
	if merged.scope.rolloutExecutionID != "" {
		scope.rolloutExecutionID = merged.scope.rolloutExecutionID
	}
	if merged.scope.changeSetID != "" {
		scope.changeSetID = merged.scope.changeSetID
	}

	category := valueOrDefault(merged.category, inferStatusCategory(auditEvent.Action))
	severity := valueOrDefault(merged.severity, inferStatusSeverity(auditEvent.Action))
	summary := valueOrDefault(merged.summary, inferStatusSummary(auditEvent.Action, details))
	automated := false
	if merged.automated != nil {
		automated = *merged.automated
	}

	metadata := merged.metadata
	if metadata == nil {
		metadata = types.Metadata{}
	}
	if len(details) > 0 {
		metadata["details"] = details
	}

	_, err = a.Status.Record(ctx, statusActorFromIdentity(identity), status.RecordRequest{
		OrganizationID:     auditEvent.OrganizationID,
		ProjectID:          scope.projectID,
		TeamID:             scope.teamID,
		ServiceID:          scope.serviceID,
		EnvironmentID:      scope.environmentID,
		RolloutExecutionID: scope.rolloutExecutionID,
		ChangeSetID:        scope.changeSetID,
		ResourceType:       auditEvent.ResourceType,
		ResourceID:         auditEvent.ResourceID,
		EventType:          auditEvent.Action,
		Category:           category,
		Severity:           severity,
		PreviousState:      merged.previousState,
		NewState:           merged.newState,
		Outcome:            auditEvent.Outcome,
		Source:             valueOrDefault(merged.source, inferStatusSource(auditEvent.Action)),
		Automated:          automated,
		Summary:            summary,
		Explanation:        compactStatusExplanation(details),
		CorrelationID:      valueOrDefault(merged.correlationID, auditEvent.ID),
		Metadata:           metadata,
	})
	return err
}

func (a *Application) resolveStatusScope(ctx context.Context, resourceType, resourceID, organizationID, projectID string) (statusScope, error) {
	scope := statusScope{projectID: projectID}
	switch resourceType {
	case "project":
		scope.projectID = resourceID
	case "team":
		team, err := a.Store.GetTeam(ctx, resourceID)
		if err != nil {
			return scope, err
		}
		scope.projectID = team.ProjectID
		scope.teamID = team.ID
	case "service":
		service, err := a.Store.GetService(ctx, resourceID)
		if err != nil {
			return scope, err
		}
		scope.projectID = service.ProjectID
		scope.serviceID = service.ID
	case "environment":
		environment, err := a.Store.GetEnvironment(ctx, resourceID)
		if err != nil {
			return scope, err
		}
		scope.projectID = environment.ProjectID
		scope.environmentID = environment.ID
	case "change_set":
		changeSet, err := a.Store.GetChangeSet(ctx, resourceID)
		if err != nil {
			return scope, err
		}
		scope.projectID = changeSet.ProjectID
		scope.serviceID = changeSet.ServiceID
		scope.environmentID = changeSet.EnvironmentID
		scope.changeSetID = changeSet.ID
	case "risk_assessment":
		assessment, err := a.Store.GetRiskAssessment(ctx, resourceID)
		if err != nil {
			return scope, err
		}
		scope.projectID = assessment.ProjectID
		scope.serviceID = assessment.ServiceID
		scope.environmentID = assessment.EnvironmentID
		scope.changeSetID = assessment.ChangeSetID
	case "rollout_plan":
		plan, err := a.Store.GetRolloutPlan(ctx, resourceID)
		if err != nil {
			return scope, err
		}
		scope.projectID = plan.ProjectID
		scope.changeSetID = plan.ChangeSetID
		if changeSet, err := a.Store.GetChangeSet(ctx, plan.ChangeSetID); err == nil {
			scope.serviceID = changeSet.ServiceID
			scope.environmentID = changeSet.EnvironmentID
		}
	case "rollout_execution":
		execution, err := a.Store.GetRolloutExecution(ctx, resourceID)
		if err != nil {
			return scope, err
		}
		scope.projectID = execution.ProjectID
		scope.serviceID = execution.ServiceID
		scope.environmentID = execution.EnvironmentID
		scope.rolloutExecutionID = execution.ID
		scope.changeSetID = execution.ChangeSetID
	case "organization", "integration", "service_account", "api_token":
		_ = organizationID
	default:
	}
	return scope, nil
}

func inferStatusCategory(eventType string) string {
	switch {
	case strings.Contains(eventType, "rollout"):
		return "rollout"
	case strings.Contains(eventType, "verification"), strings.Contains(eventType, "signal"):
		return "verification"
	case strings.Contains(eventType, "token"), strings.Contains(eventType, "service_account"), strings.Contains(eventType, "auth"):
		return "security"
	case strings.Contains(eventType, "integration"), strings.Contains(eventType, "provider"):
		return "integration"
	default:
		return "operations"
	}
}

func inferStatusSeverity(eventType string) string {
	switch {
	case strings.Contains(eventType, "failed"), strings.Contains(eventType, "denied"), strings.Contains(eventType, "rollback"):
		return "warning"
	case strings.Contains(eventType, "error"):
		return "error"
	default:
		return "info"
	}
}

func inferStatusSource(eventType string) string {
	if strings.Contains(eventType, "control_loop") || strings.Contains(eventType, "verified_automatically") {
		return "control_loop"
	}
	return "api"
}

func inferStatusSummary(eventType string, details []string) string {
	eventType = strings.TrimSpace(eventType)
	if len(details) == 0 {
		return strings.ReplaceAll(eventType, ".", " ")
	}
	return fmt.Sprintf("%s: %s", strings.ReplaceAll(eventType, ".", " "), strings.Join(details, "; "))
}

func compactStatusExplanation(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		result = append(result, strings.TrimSpace(item))
	}
	return result
}

func statusSeverityForTransition(action, nextState string) string {
	if action == "rollback" || nextState == "rolled_back" {
		return "warning"
	}
	if action == "fail" || nextState == "failed" {
		return "error"
	}
	return "info"
}

func statusSeverityForVerification(result types.VerificationResult) string {
	switch result.Decision {
	case "rollback", "failed", "advisory_rollback", "advisory_failed":
		return "warning"
	case "pause", "manual_review_required", "advisory_pause":
		return "warning"
	default:
		return "info"
	}
}

func statusActorFromIdentity(identity auth.Identity) status.Actor {
	return status.Actor{
		ID:    identity.ActorID,
		Type:  string(identity.ActorType),
		Label: identity.ActorLabel(),
	}
}

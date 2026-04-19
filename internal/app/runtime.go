package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/delivery"
	"github.com/change-control-plane/change-control-plane/internal/status"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) ClaimRolloutExecution(ctx context.Context, id string, staleBefore, claimedAt time.Time) (bool, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return false, err
	}
	execution, err := a.Store.GetRolloutExecution(ctx, id)
	if err != nil {
		return false, err
	}
	if !a.Authorizer.CanExecuteRollout(identity, execution.OrganizationID, execution.ProjectID) {
		return false, a.forbidden(ctx, identity, "rollout.execution.claim.denied", "rollout_execution", execution.ID, execution.OrganizationID, execution.ProjectID, []string{"actor lacks rollout reconciliation permission"})
	}
	return a.Store.ClaimRolloutExecution(ctx, id, staleBefore, claimedAt)
}

func (a *Application) GetRolloutExecutionRuntimeContext(ctx context.Context, id string) (types.RolloutExecutionRuntimeContext, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	execution, err := a.Store.GetRolloutExecution(ctx, id)
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	if !a.Authorizer.CanReadProject(identity, execution.OrganizationID, execution.ProjectID) {
		return types.RolloutExecutionRuntimeContext{}, ErrForbidden
	}
	plan, err := a.Store.GetRolloutPlan(ctx, execution.RolloutPlanID)
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	assessment, err := a.Store.GetRiskAssessment(ctx, plan.RiskAssessmentID)
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	changeSet, err := a.Store.GetChangeSet(ctx, execution.ChangeSetID)
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	service, err := a.Store.GetService(ctx, execution.ServiceID)
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	environment, err := a.Store.GetEnvironment(ctx, execution.EnvironmentID)
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	var backendIntegration *types.Integration
	if strings.TrimSpace(execution.BackendIntegrationID) != "" {
		integration, err := a.Store.GetIntegration(ctx, execution.BackendIntegrationID)
		if err != nil {
			return types.RolloutExecutionRuntimeContext{}, err
		}
		backendIntegration = &integration
	}
	var signalIntegration *types.Integration
	if strings.TrimSpace(execution.SignalIntegrationID) != "" {
		integration, err := a.Store.GetIntegration(ctx, execution.SignalIntegrationID)
		if err != nil {
			return types.RolloutExecutionRuntimeContext{}, err
		}
		signalIntegration = &integration
	}
	results, err := a.Store.ListVerificationResults(ctx, storage.VerificationResultQuery{
		OrganizationID:     execution.OrganizationID,
		ProjectID:          execution.ProjectID,
		RolloutExecutionID: execution.ID,
	})
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	snapshots, err := a.Store.ListSignalSnapshots(ctx, storage.SignalSnapshotQuery{
		OrganizationID:     execution.OrganizationID,
		ProjectID:          execution.ProjectID,
		RolloutExecutionID: execution.ID,
	})
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	runtimeContext := types.RolloutExecutionRuntimeContext{
		Execution:           execution,
		Plan:                plan,
		Assessment:          assessment,
		ChangeSet:           changeSet,
		Service:             service,
		Environment:         environment,
		BackendIntegration:  backendIntegration,
		SignalIntegration:   signalIntegration,
		VerificationResults: results,
		SignalSnapshots:     snapshots,
	}
	effectiveRollbackPolicy, err := a.resolveEffectiveRollbackPolicy(ctx, runtimeContext)
	if err != nil {
		return types.RolloutExecutionRuntimeContext{}, err
	}
	runtimeContext.EffectiveRollbackPolicy = effectiveRollbackPolicy
	return runtimeContext, nil
}

func (a *Application) UpdateRolloutExecutionRuntime(ctx context.Context, execution types.RolloutExecution, details []string) (types.RolloutExecution, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RolloutExecution{}, err
	}
	current, err := a.Store.GetRolloutExecution(ctx, execution.ID)
	if err != nil {
		return types.RolloutExecution{}, err
	}
	if !a.Authorizer.CanExecuteRollout(identity, current.OrganizationID, current.ProjectID) {
		return types.RolloutExecution{}, a.forbidden(ctx, identity, "rollout.execution.runtime.denied", "rollout_execution", current.ID, current.OrganizationID, current.ProjectID, []string{"actor lacks rollout runtime permission"})
	}

	execution.OrganizationID = current.OrganizationID
	execution.ProjectID = current.ProjectID
	execution.RolloutPlanID = current.RolloutPlanID
	execution.ChangeSetID = current.ChangeSetID
	execution.ServiceID = current.ServiceID
	execution.EnvironmentID = current.EnvironmentID
	execution.CreatedAt = current.CreatedAt
	execution.UpdatedAt = time.Now().UTC()
	if execution.BackendType == "" {
		execution.BackendType = current.BackendType
	}
	if execution.SignalProviderType == "" {
		execution.SignalProviderType = current.SignalProviderType
	}
	if execution.Metadata == nil {
		execution.Metadata = current.Metadata
	}

	if err := a.Store.UpdateRolloutExecution(ctx, execution); err != nil {
		return types.RolloutExecution{}, err
	}
	recordDetails := append([]string{}, details...)
	recordDetails = append(recordDetails,
		fmt.Sprintf("backend_status=%s", execution.BackendStatus),
		fmt.Sprintf("progress=%d", execution.ProgressPercent),
	)
	if err := a.record(ctx, identity, "rollout.execution.runtime_updated", "rollout_execution", execution.ID, execution.OrganizationID, execution.ProjectID, recordDetails,
		withStatusCategory("rollout"),
		withStatusSource("provider_sync"),
		withStatusSummary(fmt.Sprintf("runtime state updated to backend=%s progress=%d%%", execution.BackendStatus, execution.ProgressPercent)),
		withStatusScope(statusScope{
			projectID:          execution.ProjectID,
			serviceID:          execution.ServiceID,
			environmentID:      execution.EnvironmentID,
			rolloutExecutionID: execution.ID,
			changeSetID:        execution.ChangeSetID,
		}),
	); err != nil {
		return types.RolloutExecution{}, err
	}
	return execution, nil
}

func (a *Application) CreateSignalSnapshot(ctx context.Context, executionID string, req types.CreateSignalSnapshotRequest) (types.SignalSnapshot, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.SignalSnapshot{}, err
	}
	execution, err := a.Store.GetRolloutExecution(ctx, executionID)
	if err != nil {
		return types.SignalSnapshot{}, err
	}
	if !a.Authorizer.CanRecordVerification(identity, execution.OrganizationID, execution.ProjectID) {
		return types.SignalSnapshot{}, a.forbidden(ctx, identity, "runtime.signal.ingest.denied", "rollout_execution", execution.ID, execution.OrganizationID, execution.ProjectID, []string{"actor lacks signal ingestion permission"})
	}
	if strings.TrimSpace(req.Health) == "" {
		return types.SignalSnapshot{}, fmt.Errorf("%w: health is required", ErrValidation)
	}
	plan, err := a.Store.GetRolloutPlan(ctx, execution.RolloutPlanID)
	if err != nil {
		return types.SignalSnapshot{}, err
	}
	windowSeconds := req.WindowSeconds
	if windowSeconds <= 0 {
		windowSeconds = 300
	}
	now := time.Now().UTC()
	snapshot := types.SignalSnapshot{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("signal"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:      execution.OrganizationID,
		ProjectID:           execution.ProjectID,
		RolloutExecutionID:  execution.ID,
		RolloutPlanID:       execution.RolloutPlanID,
		ChangeSetID:         execution.ChangeSetID,
		ServiceID:           execution.ServiceID,
		EnvironmentID:       execution.EnvironmentID,
		ProviderType:        valueOrDefault(req.ProviderType, execution.SignalProviderType),
		SourceIntegrationID: strings.TrimSpace(req.SourceIntegrationID),
		Health:              strings.TrimSpace(req.Health),
		Summary:             strings.TrimSpace(req.Summary),
		Signals:             req.Signals,
		WindowStart:         now.Add(-time.Duration(windowSeconds) * time.Second),
		WindowEnd:           now,
	}
	if len(req.Explanation) > 0 {
		snapshot.Metadata = withMetadata(snapshot.Metadata, "explanation", req.Explanation)
	}
	if err := a.Store.CreateSignalSnapshot(ctx, snapshot); err != nil {
		return types.SignalSnapshot{}, err
	}
	if execution.Metadata == nil {
		execution.Metadata = types.Metadata{}
	}
	execution.Metadata["latest_signal_summary"] = snapshot.Summary
	execution.Metadata["latest_signal_health"] = snapshot.Health
	execution.LastSignalSyncAt = &now
	execution.UpdatedAt = now
	if err := a.Store.UpdateRolloutExecution(ctx, execution); err != nil {
		return types.SignalSnapshot{}, err
	}
	if err := a.record(ctx, identity, "runtime.signal.ingested", "rollout_execution", execution.ID, execution.OrganizationID, execution.ProjectID, []string{snapshot.ID, snapshot.ProviderType, snapshot.Health, plan.Strategy},
		withStatusCategory("verification"),
		withStatusSource("manual_signal_ingest"),
		withStatusSummary(fmt.Sprintf("signal snapshot %s reported %s health", snapshot.ProviderType, snapshot.Health)),
		withStatusScope(statusScope{
			projectID:          execution.ProjectID,
			serviceID:          execution.ServiceID,
			environmentID:      execution.EnvironmentID,
			rolloutExecutionID: execution.ID,
			changeSetID:        execution.ChangeSetID,
		}),
	); err != nil {
		return types.SignalSnapshot{}, err
	}
	return snapshot, nil
}

func (a *Application) ListSignalSnapshots(ctx context.Context, executionID string) ([]types.SignalSnapshot, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	execution, err := a.Store.GetRolloutExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanReadProject(identity, execution.OrganizationID, execution.ProjectID) {
		return nil, ErrForbidden
	}
	return a.Store.ListSignalSnapshots(ctx, storage.SignalSnapshotQuery{
		OrganizationID:     execution.OrganizationID,
		ProjectID:          execution.ProjectID,
		RolloutExecutionID: execution.ID,
	})
}

func (a *Application) ReconcileRolloutExecution(ctx context.Context, id string) (types.RolloutExecutionDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}
	runtimeContext, err := a.GetRolloutExecutionRuntimeContext(ctx, id)
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}
	if !a.Authorizer.CanExecuteRollout(identity, runtimeContext.Execution.OrganizationID, runtimeContext.Execution.ProjectID) {
		return types.RolloutExecutionDetail{}, a.forbidden(ctx, identity, "rollout.execution.reconcile.denied", "rollout_execution", runtimeContext.Execution.ID, runtimeContext.Execution.OrganizationID, runtimeContext.Execution.ProjectID, []string{"actor lacks rollout reconciliation permission"})
	}

	now := time.Now().UTC()
	provider, err := a.Orchestrators.Resolve(runtimeContext.Execution.BackendType)
	if err != nil {
		return types.RolloutExecutionDetail{}, a.persistRuntimeFailure(ctx, identity, runtimeContext.Execution, fmt.Sprintf("provider resolution failed: %v", err))
	}

	action, syncResult, err := a.performProviderSync(ctx, runtimeContext, provider)
	if err != nil {
		return types.RolloutExecutionDetail{}, a.persistRuntimeFailure(ctx, identity, runtimeContext.Execution, fmt.Sprintf("%s failed: %v", action, err))
	}
	updatedExecution, err := a.persistProviderResult(ctx, identity, runtimeContext.Execution, action, syncResult, now)
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}
	_ = updatedExecution

	runtimeContext, err = a.GetRolloutExecutionRuntimeContext(ctx, id)
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}

	signalProvider, err := a.Signals.Resolve(runtimeContext.Execution.SignalProviderType)
	if err != nil {
		return types.RolloutExecutionDetail{}, a.persistRuntimeFailure(ctx, identity, runtimeContext.Execution, fmt.Sprintf("signal provider resolution failed: %v", err))
	}
	collection, err := signalProvider.Collect(ctx, runtimeContext)
	if err != nil {
		return types.RolloutExecutionDetail{}, a.persistRuntimeFailure(ctx, identity, runtimeContext.Execution, fmt.Sprintf("signal collection failed: %v", err))
	}
	if len(collection.Snapshots) > 0 {
		for _, snapshot := range collection.Snapshots {
			if err := a.Store.CreateSignalSnapshot(ctx, snapshot); err != nil {
				return types.RolloutExecutionDetail{}, err
			}
		}
		latest := collection.Snapshots[len(collection.Snapshots)-1]
		runtimeContext.Execution.LastSignalSyncAt = &now
		runtimeContext.Execution.Metadata = mergeRuntimeMetadata(runtimeContext.Execution.Metadata, types.Metadata{
			"latest_signal_summary": latest.Summary,
			"latest_signal_health":  latest.Health,
		})
		runtimeContext.Execution.UpdatedAt = now
		if err := a.Store.UpdateRolloutExecution(ctx, runtimeContext.Execution); err != nil {
			return types.RolloutExecutionDetail{}, err
		}
		if err := a.record(ctx, identity, "runtime.signal.collected", "rollout_execution", runtimeContext.Execution.ID, runtimeContext.Execution.OrganizationID, runtimeContext.Execution.ProjectID, compactDetailList(append([]string{collection.Source}, collection.Explanation...)),
			withStatusCategory("verification"),
			withStatusSource(collection.Source),
			withStatusAutomated(true),
			withStatusSummary(fmt.Sprintf("collected %d signal snapshot(s) from %s", len(collection.Snapshots), collection.Source)),
			withStatusScope(statusScope{
				projectID:          runtimeContext.Execution.ProjectID,
				serviceID:          runtimeContext.Execution.ServiceID,
				environmentID:      runtimeContext.Execution.EnvironmentID,
				rolloutExecutionID: runtimeContext.Execution.ID,
				changeSetID:        runtimeContext.Execution.ChangeSetID,
			}),
		); err != nil {
			return types.RolloutExecutionDetail{}, err
		}
		runtimeContext, err = a.GetRolloutExecutionRuntimeContext(ctx, id)
		if err != nil {
			return types.RolloutExecutionDetail{}, err
		}
	}

	evaluation := a.Verifier.Evaluate(runtimeContext, syncResult)
	if evaluation.Record {
		if advisoryOnlyRuntime(runtimeContext) {
			evaluation.Request = advisoryVerificationRequest(evaluation.Request)
		}
		result, err := a.RecordVerificationResult(ctx, runtimeContext.Execution.ID, evaluation.Request)
		if err != nil {
			return types.RolloutExecutionDetail{}, err
		}
		if err := a.record(ctx, identity, "rollout.execution.verified_automatically", "rollout_execution", runtimeContext.Execution.ID, runtimeContext.Execution.OrganizationID, runtimeContext.Execution.ProjectID, compactDetailList(evaluation.Explanation),
			withStatusCategory("verification"),
			withStatusAutomated(true),
			withStatusSource(valueOrDefault(result.DecisionSource, "control_loop")),
			withStatusSeverity(statusSeverityForVerification(result)),
			withStatusSummary(result.Summary),
			withStatusScope(statusScope{
				projectID:          runtimeContext.Execution.ProjectID,
				serviceID:          runtimeContext.Execution.ServiceID,
				environmentID:      runtimeContext.Execution.EnvironmentID,
				rolloutExecutionID: runtimeContext.Execution.ID,
				changeSetID:        runtimeContext.Execution.ChangeSetID,
			}),
		); err != nil {
			return types.RolloutExecutionDetail{}, err
		}
		runtimeContext, err = a.GetRolloutExecutionRuntimeContext(ctx, id)
		if err != nil {
			return types.RolloutExecutionDetail{}, err
		}
		followupAction, followupResult, err := a.performProviderSync(ctx, runtimeContext, provider)
		if err != nil {
			return types.RolloutExecutionDetail{}, a.persistRuntimeFailure(ctx, identity, runtimeContext.Execution, fmt.Sprintf("post-verification action failed: %v", err))
		}
		if followupAction != "sync" || followupResult.BackendStatus != runtimeContext.Execution.BackendStatus {
			if _, err := a.persistProviderResult(ctx, identity, runtimeContext.Execution, followupAction, followupResult, time.Now().UTC()); err != nil {
				return types.RolloutExecutionDetail{}, err
			}
		}
	}

	return a.GetRolloutExecutionDetail(ctx, id)
}

func (a *Application) persistRuntimeFailure(ctx context.Context, identity auth.Identity, execution types.RolloutExecution, message string) error {
	execution.LastError = message
	execution.UpdatedAt = time.Now().UTC()
	_, err := a.UpdateRolloutExecutionRuntime(ctx, execution, []string{"runtime_error", message})
	if err != nil {
		return err
	}
	_, _ = a.Audit.Record(ctx, auditActorFromIdentity(identity), "rollout.execution.reconcile_failed", "rollout_execution", execution.ID, "error", execution.OrganizationID, execution.ProjectID, []string{message})
	_, _ = a.Status.Record(ctx, statusActorFromIdentity(identity), status.RecordRequest{
		OrganizationID:     execution.OrganizationID,
		ProjectID:          execution.ProjectID,
		ServiceID:          execution.ServiceID,
		EnvironmentID:      execution.EnvironmentID,
		RolloutExecutionID: execution.ID,
		ChangeSetID:        execution.ChangeSetID,
		ResourceType:       "rollout_execution",
		ResourceID:         execution.ID,
		EventType:          "rollout.execution.reconcile_failed",
		Category:           "rollout",
		Severity:           "error",
		Outcome:            "error",
		Source:             "control_loop",
		Automated:          true,
		Summary:            message,
		Explanation:        []string{message},
	})
	return fmt.Errorf("%w: %s", ErrValidation, message)
}

func (a *Application) performProviderSync(ctx context.Context, runtimeContext types.RolloutExecutionRuntimeContext, provider delivery.Provider) (string, delivery.SyncResult, error) {
	action := "sync"
	switch {
	case runtimeContext.Execution.Status == "in_progress" && strings.TrimSpace(runtimeContext.Execution.BackendExecutionID) == "":
		action = "submit"
	case runtimeContext.Execution.Status == "paused" && runtimeContext.Execution.BackendStatus != "paused":
		action = "pause"
	case runtimeContext.Execution.Status == "rolled_back" && runtimeContext.Execution.BackendStatus != "rolled_back" && runtimeContext.Execution.BackendStatus != "rollback_requested" && runtimeContext.Execution.BackendStatus != "rollback_in_progress":
		action = "rollback"
	case runtimeContext.Execution.Status == "in_progress" && runtimeContext.Execution.BackendStatus == "paused":
		action = "resume"
	}
	if advisoryOnlyRuntime(runtimeContext) && action != "sync" {
		observed, observeErr := provider.Sync(ctx, runtimeContext)
		if observeErr != nil {
			return action, delivery.SyncResult{}, observeErr
		}
		if observed.Metadata == nil {
			observed.Metadata = types.Metadata{}
		}
		observed.Metadata["advisory_only"] = true
		observed.Metadata["recommended_action"] = action
		observed.Metadata["action_disposition"] = "suppressed"
		observed.Metadata["control_mode"] = "advisory"
		observed.Metadata["control_rationale"] = "active control is disabled for the configured backend integration"
		observed.Summary = fmt.Sprintf("advisory mode observed backend without executing %s", action)
		observed.Explanation = compactDetailList(append(observed.Explanation, "active control is disabled for the configured backend integration"))
		return "sync", observed, nil
	}
	switch action {
	case "submit":
		syncResult, err := provider.Submit(ctx, runtimeContext)
		return action, syncResult, err
	case "pause":
		syncResult, err := provider.Pause(ctx, runtimeContext, runtimeContext.Execution.LastDecisionReason)
		return action, syncResult, err
	case "rollback":
		syncResult, err := provider.Rollback(ctx, runtimeContext, runtimeContext.Execution.LastDecisionReason)
		return action, syncResult, err
	case "resume":
		syncResult, err := provider.Resume(ctx, runtimeContext, runtimeContext.Execution.LastDecisionReason)
		return action, syncResult, err
	default:
		syncResult, err := provider.Sync(ctx, runtimeContext)
		return action, syncResult, err
	}
}

func advisoryOnlyRuntime(runtimeContext types.RolloutExecutionRuntimeContext) bool {
	if runtimeContext.BackendIntegration == nil {
		return false
	}
	if strings.EqualFold(runtimeContext.Execution.BackendType, "simulated") {
		return false
	}
	return !integrationAllowsActiveControl(runtimeContext.BackendIntegration)
}

func advisoryVerificationRequest(request types.RecordVerificationResultRequest) types.RecordVerificationResultRequest {
	updated := request
	switch strings.TrimSpace(request.Decision) {
	case "rollback":
		updated.Decision = "advisory_rollback"
	case "pause", "manual_review_required":
		updated.Decision = "advisory_pause"
	case "failed":
		updated.Decision = "advisory_failed"
	default:
		updated.Decision = "advisory_verified"
	}
	updated.Summary = "Advisory recommendation: " + strings.TrimSpace(request.Summary)
	updated.Explanation = compactDetailList(append(updated.Explanation, "external deployment control is disabled; the control plane recorded an advisory recommendation instead of acting on the deployment target"))
	return updated
}

func (a *Application) persistProviderResult(ctx context.Context, identity auth.Identity, execution types.RolloutExecution, action string, syncResult delivery.SyncResult, now time.Time) (types.RolloutExecution, error) {
	updatedExecution := execution
	disposition := providerActionDisposition(action, syncResult)
	recordedAction := recordedProviderAction(action, syncResult)
	recommendedAction := metadataString(syncResult.Metadata, "recommended_action")
	controlMode := metadataString(syncResult.Metadata, "control_mode")
	controlRationale := metadataString(syncResult.Metadata, "control_rationale")
	if controlMode == "" && strings.TrimSpace(execution.BackendIntegrationID) != "" && !strings.EqualFold(strings.TrimSpace(execution.BackendType), "simulated") {
		controlMode = "active_control"
	}
	updatedExecution.BackendType = valueOrDefault(syncResult.BackendType, updatedExecution.BackendType)
	updatedExecution.BackendExecutionID = valueOrDefault(syncResult.BackendExecutionID, updatedExecution.BackendExecutionID)
	updatedExecution.BackendStatus = valueOrDefault(syncResult.BackendStatus, updatedExecution.BackendStatus)
	if syncResult.ProgressPercent > 0 || updatedExecution.ProgressPercent == 0 {
		updatedExecution.ProgressPercent = syncResult.ProgressPercent
	}
	updatedExecution.CurrentStep = valueOrDefault(syncResult.CurrentStep, updatedExecution.CurrentStep)
	updatedExecution.LastBackendSyncAt = &now
	updatedExecution.LastError = ""
	updatedExecution.Metadata = mergeRuntimeMetadata(updatedExecution.Metadata, types.Metadata{
		"backend_summary":         syncResult.Summary,
		"backend_explanation":     syncResult.Explanation,
		"last_provider_action":    recordedAction,
		"last_action_disposition": disposition,
		"recommended_action":      recommendedAction,
		"control_mode":            controlMode,
		"control_rationale":       controlRationale,
	})
	if action == "submit" && updatedExecution.SubmittedAt == nil {
		updatedExecution.SubmittedAt = &now
	}
	if action == "rollback" && updatedExecution.Metadata == nil {
		updatedExecution.Metadata = types.Metadata{}
	}
	if action == "rollback" {
		updatedExecution.Metadata["rollback_requested_at"] = now.Format(time.RFC3339)
	}
	if _, err := a.UpdateRolloutExecutionRuntime(ctx, updatedExecution, compactDetailList([]string{
		fmt.Sprintf("action=%s", recordedAction),
		fmt.Sprintf("action_disposition=%s", disposition),
		fmt.Sprintf("backend=%s", updatedExecution.BackendType),
		fmt.Sprintf("backend_status=%s", updatedExecution.BackendStatus),
		syncResult.Summary,
	})); err != nil {
		return types.RolloutExecution{}, err
	}
	if err := a.recordProviderActionStatus(ctx, identity, updatedExecution, recordedAction, disposition, syncResult); err != nil {
		return types.RolloutExecution{}, err
	}
	return updatedExecution, nil
}

func providerActionDisposition(action string, syncResult delivery.SyncResult) string {
	if value := metadataString(syncResult.Metadata, "action_disposition"); value != "" {
		return value
	}
	if strings.TrimSpace(action) == "sync" {
		return "observed"
	}
	return "executed"
}

func recordedProviderAction(action string, syncResult delivery.SyncResult) string {
	if value := metadataString(syncResult.Metadata, "recommended_action"); value != "" {
		return value
	}
	return strings.TrimSpace(action)
}

func (a *Application) recordProviderActionStatus(ctx context.Context, identity auth.Identity, execution types.RolloutExecution, action, disposition string, syncResult delivery.SyncResult) error {
	if action == "" || action == "sync" {
		return nil
	}
	scope := statusScope{
		projectID:          execution.ProjectID,
		serviceID:          execution.ServiceID,
		environmentID:      execution.EnvironmentID,
		rolloutExecutionID: execution.ID,
		changeSetID:        execution.ChangeSetID,
	}
	switch disposition {
	case "suppressed":
		_, err := a.Status.Record(ctx, status.Actor{}, status.RecordRequest{
			OrganizationID:     execution.OrganizationID,
			ProjectID:          execution.ProjectID,
			ServiceID:          execution.ServiceID,
			EnvironmentID:      execution.EnvironmentID,
			RolloutExecutionID: execution.ID,
			ChangeSetID:        execution.ChangeSetID,
			ResourceType:       "rollout_execution",
			ResourceID:         execution.ID,
			EventType:          "rollout.execution.action_suppressed",
			Category:           "rollout",
			Severity:           "warning",
			Outcome:            "suppressed",
			Source:             "control_loop",
			Automated:          true,
			Summary:            fmt.Sprintf("advisory mode recommended %s but did not execute it", action),
			Explanation: compactDetailList([]string{
				syncResult.Summary,
				"external deployment control is disabled for the configured backend integration",
			}),
			Metadata: types.Metadata{
				"recommended_action": action,
				"action_disposition": disposition,
			},
		})
		if err != nil {
			return err
		}
		return nil
	case "executed":
		return a.record(ctx, identity, "rollout.execution.action_executed", "rollout_execution", execution.ID, execution.OrganizationID, execution.ProjectID, compactDetailList([]string{
			action,
			syncResult.Summary,
		}),
			withStatusCategory("rollout"),
			withStatusSource("control_loop"),
			withStatusAutomated(true),
			withStatusSeverity(statusSeverityForTransition(action, execution.Status)),
			withStatusSummary(fmt.Sprintf("executed external %s against the backend provider", action)),
			withStatusScope(scope),
		)
	default:
		return nil
	}
}

func mergeRuntimeMetadata(current, additions types.Metadata) types.Metadata {
	if current == nil && len(additions) == 0 {
		return nil
	}
	result := types.Metadata{}
	for key, value := range current {
		result[key] = value
	}
	for key, value := range additions {
		if value == nil {
			continue
		}
		result[key] = value
	}
	return result
}

func compactDetailList(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		result = append(result, strings.TrimSpace(item))
	}
	return result
}

func currentIdentity(ctx context.Context) (auth.Identity, bool) {
	return auth.IdentityFromContext(ctx)
}

package delivery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type SimulatedProvider struct{}

func NewSimulatedProvider() Provider {
	return SimulatedProvider{}
}

func (SimulatedProvider) Kind() string {
	return "simulated"
}

func (SimulatedProvider) Submit(_ context.Context, runtime types.RolloutExecutionRuntimeContext) (SyncResult, error) {
	now := time.Now().UTC()
	executionID := runtime.Execution.BackendExecutionID
	if strings.TrimSpace(executionID) == "" {
		executionID = fmt.Sprintf("simexec_%s", runtime.Execution.ID)
	}
	return SyncResult{
		BackendType:        "simulated",
		BackendExecutionID: executionID,
		BackendStatus:      "queued",
		ProgressPercent:    maxInt(runtime.Execution.ProgressPercent, 10),
		CurrentStep:        fallbackStep(runtime.Execution.CurrentStep, "submission"),
		Summary:            "simulated orchestrator accepted rollout execution",
		Explanation: []string{
			"simulated backend created a stable execution handle for the rollout",
			"execution is queued and ready for deterministic progress reconciliation",
		},
		LastUpdatedAt: now,
	}, nil
}

func (SimulatedProvider) Sync(_ context.Context, runtime types.RolloutExecutionRuntimeContext) (SyncResult, error) {
	now := time.Now().UTC()
	latestSnapshot, hasSnapshot := latestSignalSnapshot(runtime.SignalSnapshots)
	progress := runtime.Execution.ProgressPercent
	if progress <= 0 {
		progress = 10
	}

	result := SyncResult{
		BackendType:        "simulated",
		BackendExecutionID: fallbackExecutionID(runtime.Execution.BackendExecutionID, runtime.Execution.ID),
		BackendStatus:      runtime.Execution.BackendStatus,
		ProgressPercent:    progress,
		CurrentStep:        runtime.Execution.CurrentStep,
		LastUpdatedAt:      now,
	}

	switch runtime.Execution.Status {
	case "awaiting_approval":
		result.BackendStatus = fallbackStatus(runtime.Execution.BackendStatus, "waiting_for_approval")
		result.CurrentStep = fallbackStep(runtime.Execution.CurrentStep, "approval")
		result.Summary = "rollout is waiting for approval before submission"
		result.Explanation = []string{"simulated backend will not submit the rollout until the control plane approves it"}
	case "planned":
		result.BackendStatus = fallbackStatus(runtime.Execution.BackendStatus, "pending_submission")
		result.CurrentStep = fallbackStep(runtime.Execution.CurrentStep, "queued")
		result.ProgressPercent = maxInt(progress, 5)
		result.Summary = "rollout is planned and awaiting control-plane start"
		result.Explanation = []string{"simulated backend is idle until the control plane transitions the rollout into progress"}
	case "approved", "in_progress":
		if hasSnapshot {
			result.BackendStatus = "awaiting_verification"
			result.ProgressPercent = maxInt(progress, 80)
			result.CurrentStep = "verification_gate"
			result.Summary = fmt.Sprintf("simulated backend reached verification gate with %s runtime health", latestSnapshot.Health)
			result.Explanation = []string{
				"deployment steps completed and the execution is waiting on runtime verification",
				fmt.Sprintf("latest simulated signal snapshot is %s", latestSnapshot.Health),
			}
		} else {
			result.BackendStatus = "progressing"
			result.ProgressPercent = maxInt(progress, 65)
			result.CurrentStep = "deploying"
			result.Summary = "simulated backend is progressing rollout steps toward the verification checkpoint"
			result.Explanation = []string{
				"deployment is advancing through deterministic simulated stages",
				"no runtime signal snapshot has been ingested yet, so the execution remains in progress",
			}
		}
	case "paused":
		result.BackendStatus = "paused"
		result.ProgressPercent = maxInt(progress, 80)
		result.CurrentStep = "paused"
		result.Summary = "simulated backend paused rollout execution"
		result.Explanation = []string{"rollout is paused and awaiting manual resume or rollback"}
	case "verified":
		result.BackendStatus = "succeeded"
		result.ProgressPercent = 100
		result.CurrentStep = "verification_passed"
		result.Summary = "simulated backend marked rollout verified and safe to complete"
		result.Explanation = []string{"verification passed and the simulated backend considers the rollout healthy"}
	case "completed":
		result.BackendStatus = "succeeded"
		result.ProgressPercent = 100
		result.CurrentStep = "completed"
		result.Summary = "simulated backend completed rollout execution"
		result.Explanation = []string{"rollout finished successfully in the simulated backend"}
	case "rolled_back":
		result.BackendStatus = "rolled_back"
		result.ProgressPercent = 100
		result.CurrentStep = "rollback_complete"
		result.Summary = "simulated backend rolled back rollout execution"
		result.Explanation = []string{"rollback completed and the execution is terminal"}
	case "failed":
		result.BackendStatus = "failed"
		result.ProgressPercent = maxInt(progress, 100)
		result.CurrentStep = "failed"
		result.Summary = "simulated backend failed rollout execution"
		result.Explanation = []string{"execution reached a terminal failed state in the simulated backend"}
	default:
		result.BackendStatus = fallbackStatus(runtime.Execution.BackendStatus, "unknown")
		result.CurrentStep = fallbackStep(runtime.Execution.CurrentStep, "unknown")
		result.Summary = "simulated backend returned the stored execution state without change"
	}

	return result, nil
}

func (SimulatedProvider) Pause(_ context.Context, runtime types.RolloutExecutionRuntimeContext, reason string) (SyncResult, error) {
	return SyncResult{
		BackendType:        "simulated",
		BackendExecutionID: fallbackExecutionID(runtime.Execution.BackendExecutionID, runtime.Execution.ID),
		BackendStatus:      "paused",
		ProgressPercent:    maxInt(runtime.Execution.ProgressPercent, 80),
		CurrentStep:        "paused",
		Summary:            "simulated backend applied a pause request",
		Explanation:        compactStrings([]string{"control plane requested a pause", reason}),
		LastUpdatedAt:      time.Now().UTC(),
	}, nil
}

func (SimulatedProvider) Resume(_ context.Context, runtime types.RolloutExecutionRuntimeContext, reason string) (SyncResult, error) {
	return SyncResult{
		BackendType:        "simulated",
		BackendExecutionID: fallbackExecutionID(runtime.Execution.BackendExecutionID, runtime.Execution.ID),
		BackendStatus:      "progressing",
		ProgressPercent:    maxInt(runtime.Execution.ProgressPercent, 65),
		CurrentStep:        "deploying",
		Summary:            "simulated backend resumed rollout execution",
		Explanation:        compactStrings([]string{"control plane requested a resume", reason}),
		LastUpdatedAt:      time.Now().UTC(),
	}, nil
}

func (SimulatedProvider) Rollback(_ context.Context, runtime types.RolloutExecutionRuntimeContext, reason string) (SyncResult, error) {
	return SyncResult{
		BackendType:        "simulated",
		BackendExecutionID: fallbackExecutionID(runtime.Execution.BackendExecutionID, runtime.Execution.ID),
		BackendStatus:      "rolled_back",
		ProgressPercent:    100,
		CurrentStep:        "rollback_complete",
		Summary:            "simulated backend rolled back the execution",
		Explanation:        compactStrings([]string{"control plane requested a rollback", reason}),
		LastUpdatedAt:      time.Now().UTC(),
	}, nil
}

func latestSignalSnapshot(items []types.SignalSnapshot) (types.SignalSnapshot, bool) {
	if len(items) == 0 {
		return types.SignalSnapshot{}, false
	}
	return items[len(items)-1], true
}

func fallbackExecutionID(existing, executionID string) string {
	if strings.TrimSpace(existing) != "" {
		return existing
	}
	return fmt.Sprintf("simexec_%s", executionID)
}

func fallbackStatus(existing, fallback string) string {
	if strings.TrimSpace(existing) != "" {
		return existing
	}
	return fallback
}

func fallbackStep(existing, fallback string) string {
	if strings.TrimSpace(existing) != "" {
		return existing
	}
	return fallback
}

func compactStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		result = append(result, strings.TrimSpace(item))
	}
	return result
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

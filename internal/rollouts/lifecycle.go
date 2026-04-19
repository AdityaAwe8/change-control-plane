package rollouts

import (
	"fmt"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func InitialExecutionStatus(plan types.RolloutPlan) string {
	if plan.ApprovalRequired {
		return "awaiting_approval"
	}
	return "planned"
}

func InitialExecutionStep(plan types.RolloutPlan) string {
	if len(plan.Steps) == 0 {
		return ""
	}
	return plan.Steps[0].Name
}

func AdvanceExecution(execution types.RolloutExecution, action, reason string, now time.Time) (types.RolloutExecution, error) {
	next, err := nextExecutionState(execution.Status, action)
	if err != nil {
		return execution, err
	}
	execution.Status = next
	execution.LastDecision = action
	execution.LastDecisionReason = reason

	if action == "start" && execution.StartedAt == nil {
		execution.StartedAt = &now
	}
	if next == "completed" || next == "rolled_back" || next == "failed" {
		execution.CompletedAt = &now
	}
	if next == "in_progress" && execution.StartedAt == nil {
		execution.StartedAt = &now
	}
	return execution, nil
}

func ApplyVerificationDecision(execution types.RolloutExecution, result types.VerificationResult, now time.Time) (types.RolloutExecution, error) {
	execution.LastVerificationResult = result.ID
	execution.LastDecision = result.Decision
	execution.LastDecisionReason = result.Summary

	switch result.Decision {
	case "continue", "verified":
		execution.Status = "verified"
	case "pause", "manual_review_required":
		execution.Status = "paused"
	case "rollback":
		execution.Status = "rolled_back"
		execution.CompletedAt = &now
	case "failed":
		execution.Status = "failed"
		execution.CompletedAt = &now
	case "advisory_continue", "advisory_verified", "advisory_pause", "advisory_rollback", "advisory_failed":
		// Advisory decisions record evidence without changing the desired rollout state.
	default:
		return execution, fmt.Errorf("unsupported verification decision %q", result.Decision)
	}
	return execution, nil
}

func nextExecutionState(current, action string) (string, error) {
	transitions := map[string]map[string]string{
		"planned": {
			"start": "in_progress",
			"pause": "paused",
		},
		"awaiting_approval": {
			"approve":  "approved",
			"rollback": "rolled_back",
		},
		"approved": {
			"start": "in_progress",
			"pause": "paused",
		},
		"in_progress": {
			"pause":    "paused",
			"verify":   "verified",
			"fail":     "failed",
			"rollback": "rolled_back",
			"complete": "completed",
		},
		"paused": {
			"continue": "in_progress",
			"resume":   "in_progress",
			"rollback": "rolled_back",
			"fail":     "failed",
		},
		"verified": {
			"continue": "in_progress",
			"complete": "completed",
			"rollback": "rolled_back",
			"pause":    "paused",
		},
		"failed": {
			"rollback": "rolled_back",
		},
	}
	if next, ok := transitions[current][action]; ok {
		return next, nil
	}
	return "", fmt.Errorf("invalid rollout transition %q from %q", action, current)
}

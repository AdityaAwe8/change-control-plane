package verification

import (
	"fmt"
	"slices"
	"strings"

	"github.com/change-control-plane/change-control-plane/internal/delivery"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Engine struct{}

type Evaluation struct {
	Record      bool                                  `json:"record"`
	Request     types.RecordVerificationResultRequest `json:"request"`
	Explanation []string                              `json:"explanation,omitempty"`
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Evaluate(runtime types.RolloutExecutionRuntimeContext, backend delivery.SyncResult) Evaluation {
	if runtime.Execution.Status != "in_progress" && runtime.Execution.Status != "paused" && runtime.Execution.Status != "verified" {
		return Evaluation{}
	}

	latestResult, hasResult := latestVerification(runtime.VerificationResults)
	latestSnapshot, hasSnapshot := latestSignal(runtime.SignalSnapshots)
	policy := effectiveRollbackPolicy(runtime)

	if backend.BackendStatus == "failed" {
		request := types.RecordVerificationResultRequest{
			Outcome:        "fail",
			Automated:      true,
			DecisionSource: "control_loop",
			Signals:        []string{"backend_status"},
			Metadata: types.Metadata{
				"backend_status": backend.BackendStatus,
				"backend_type":   backend.BackendType,
				"policy_id":      policy.ID,
				"policy_name":    policy.Name,
			},
		}
		if policy.RollbackOnProviderFailure {
			request.Decision = "rollback"
			request.Summary = "provider failure triggered automatic rollback"
			request.Explanation = compactStrings([]string{
				"backend execution reached a failed state",
				backend.Summary,
				fmt.Sprintf("rollback policy %q requires rollback on provider failure", policy.Name),
			})
		} else {
			request.Decision = "failed"
			request.Summary = "orchestrator reported a terminal failed rollout state"
			request.Explanation = compactStrings([]string{
				"backend execution reached a failed state",
				backend.Summary,
				fmt.Sprintf("rollback policy %q does not automatically rollback provider failures", policy.Name),
			})
		}
		if duplicateDecision(latestResult, hasResult, request) {
			return Evaluation{}
		}
		return Evaluation{Record: true, Request: request, Explanation: request.Explanation}
	}

	if backend.BackendStatus != "awaiting_verification" && backend.BackendStatus != "succeeded" {
		return Evaluation{}
	}

	if !hasSnapshot {
		if len(runtime.Plan.VerificationSignals) == 0 {
			request := types.RecordVerificationResultRequest{
				Outcome:        "pass",
				Decision:       "verified",
				Summary:        "rollout reached completion without additional verification signals",
				Explanation:    compactStrings([]string{"rollout plan did not require explicit runtime verification signals", backend.Summary}),
				Automated:      true,
				DecisionSource: "control_loop",
				Signals:        []string{"backend_status"},
				Metadata: types.Metadata{
					"backend_status": backend.BackendStatus,
					"backend_type":   backend.BackendType,
					"policy_id":      policy.ID,
					"policy_name":    policy.Name,
				},
			}
			if duplicateDecision(latestResult, hasResult, request) {
				return Evaluation{}
			}
			return Evaluation{Record: true, Request: request, Explanation: request.Explanation}
		}
		return Evaluation{}
	}

	decision := types.RecordVerificationResultRequest{
		Automated:         true,
		DecisionSource:    "control_loop",
		SignalSnapshotIDs: []string{latestSnapshot.ID},
		Signals:           signalNames(latestSnapshot.Signals),
		TechnicalSignalSummary: types.Metadata{
			"health":       latestSnapshot.Health,
			"summary":      latestSnapshot.Summary,
			"provider":     latestSnapshot.ProviderType,
			"window_start": latestSnapshot.WindowStart.Format(timeLayout),
			"window_end":   latestSnapshot.WindowEnd.Format(timeLayout),
		},
	}

	decision.BusinessSignalSummary = businessSummary(latestSnapshot.Signals)
	decision.Metadata = types.Metadata{
		"backend_status": backend.BackendStatus,
		"backend_type":   backend.BackendType,
		"policy_id":      policy.ID,
		"policy_name":    policy.Name,
	}
	breaches := policyBreaches(latestSnapshot, policy)
	if len(breaches) > 0 {
		decision.Metadata["policy_breaches"] = breaches
	}
	failureCount := verificationFailureCount(runtime.VerificationResults)
	effectiveHealth := effectiveSnapshotHealth(latestSnapshot.Health, breaches)

	switch strings.ToLower(strings.TrimSpace(effectiveHealth)) {
	case "healthy", "pass":
		decision.Outcome = "pass"
		decision.Decision = "verified"
		decision.Summary = "runtime verification passed"
		decision.Explanation = compactStrings([]string{
			fmt.Sprintf("latest signal snapshot reported %s health", effectiveHealth),
			backend.Summary,
			"deterministic guardrails remained within tolerated thresholds",
		})
	case "warning", "degraded":
		decision.Outcome = "inconclusive"
		if policy.MaxVerificationFailures > 0 && failureCount+1 >= policy.MaxVerificationFailures {
			decision.Decision = "rollback"
			decision.Summary = "runtime verification exceeded rollback policy retry budget"
			decision.Explanation = compactStrings([]string{
				fmt.Sprintf("latest signal snapshot reported %s health", effectiveHealth),
				fmt.Sprintf("verification failure count %d reached policy limit %d", failureCount+1, policy.MaxVerificationFailures),
				backend.Summary,
			})
		} else {
			decision.Decision = "manual_review_required"
			decision.Summary = "runtime verification requires manual review"
			decision.Explanation = compactStrings([]string{
				fmt.Sprintf("latest signal snapshot reported %s health", effectiveHealth),
				"guardrails show elevated risk but not a terminal failure condition",
				backend.Summary,
			})
		}
	case "critical", "fail", "unhealthy":
		decision.Outcome = "fail"
		if policy.RollbackOnCriticalSignals || (policy.MaxVerificationFailures > 0 && failureCount+1 >= policy.MaxVerificationFailures) {
			decision.Decision = "rollback"
			decision.Summary = "runtime verification triggered rollback"
			decision.Explanation = compactStrings([]string{
				fmt.Sprintf("latest signal snapshot reported %s health", effectiveHealth),
				fmt.Sprintf("rollback policy %q requires rollback for this failure condition", policy.Name),
				backend.Summary,
				strings.Join(breaches, "; "),
			})
		} else {
			decision.Decision = "pause"
			decision.Summary = "runtime verification paused rollout"
			decision.Explanation = compactStrings([]string{
				fmt.Sprintf("latest signal snapshot reported %s health", effectiveHealth),
				"control plane paused the rollout for investigation because rollback was not mandatory",
				backend.Summary,
				strings.Join(breaches, "; "),
			})
		}
	default:
		return Evaluation{}
	}

	if duplicateDecision(latestResult, hasResult, decision) {
		return Evaluation{}
	}
	return Evaluation{Record: true, Request: decision, Explanation: decision.Explanation}
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func latestVerification(items []types.VerificationResult) (types.VerificationResult, bool) {
	if len(items) == 0 {
		return types.VerificationResult{}, false
	}
	return items[len(items)-1], true
}

func latestSignal(items []types.SignalSnapshot) (types.SignalSnapshot, bool) {
	if len(items) == 0 {
		return types.SignalSnapshot{}, false
	}
	return items[len(items)-1], true
}

func signalNames(items []types.SignalValue) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		names = append(names, item.Name)
	}
	return names
}

func businessSummary(items []types.SignalValue) types.Metadata {
	result := types.Metadata{}
	for _, item := range items {
		if !strings.EqualFold(item.Category, "business") {
			continue
		}
		result[item.Name] = types.Metadata{
			"value":      item.Value,
			"status":     item.Status,
			"threshold":  item.Threshold,
			"comparator": item.Comparator,
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func duplicateDecision(latest types.VerificationResult, ok bool, proposed types.RecordVerificationResultRequest) bool {
	if !ok || !latest.Automated {
		return false
	}
	if latest.DecisionSource != proposed.DecisionSource {
		return false
	}
	if latest.Decision != proposed.Decision || latest.Outcome != proposed.Outcome || latest.Summary != proposed.Summary {
		return false
	}
	return slices.Equal(latest.SignalSnapshotIDs, proposed.SignalSnapshotIDs)
}

func shouldRollback(runtime types.RolloutExecutionRuntimeContext) bool {
	if runtime.Environment.Production {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(string(runtime.Assessment.Level))) {
	case string(types.RiskLevelHigh), string(types.RiskLevelCritical):
		return true
	}
	if strings.EqualFold(runtime.Service.Criticality, "mission_critical") || runtime.Service.CustomerFacing {
		return true
	}
	return false
}

func effectiveRollbackPolicy(runtime types.RolloutExecutionRuntimeContext) types.RollbackPolicy {
	if runtime.EffectiveRollbackPolicy != nil {
		return *runtime.EffectiveRollbackPolicy
	}
	return types.RollbackPolicy{
		Name:                      "fallback policy",
		RollbackOnProviderFailure: true,
		RollbackOnCriticalSignals: shouldRollback(runtime),
		MaxVerificationFailures:   1,
	}
}

func effectiveSnapshotHealth(snapshotHealth string, breaches []string) string {
	health := strings.ToLower(strings.TrimSpace(snapshotHealth))
	if len(breaches) == 0 {
		return health
	}
	switch health {
	case "", "healthy", "pass", "warning", "degraded":
		return "critical"
	default:
		return health
	}
}

func verificationFailureCount(items []types.VerificationResult) int {
	count := 0
	for _, item := range items {
		switch item.Decision {
		case "pause", "rollback", "failed", "manual_review_required":
			count++
		}
	}
	return count
}

func policyBreaches(snapshot types.SignalSnapshot, policy types.RollbackPolicy) []string {
	breaches := []string{}
	for _, signal := range snapshot.Signals {
		name := strings.ToLower(strings.TrimSpace(signal.Name))
		switch {
		case name == "error_rate" && policy.MaxErrorRate > 0 && signal.Value > policy.MaxErrorRate:
			breaches = append(breaches, fmt.Sprintf("error rate %0.3f exceeded max %0.3f", signal.Value, policy.MaxErrorRate))
		case (name == "latency_ms" || name == "latency_p95_ms") && policy.MaxLatencyMs > 0 && signal.Value > policy.MaxLatencyMs:
			breaches = append(breaches, fmt.Sprintf("latency %0.3fms exceeded max %0.3fms", signal.Value, policy.MaxLatencyMs))
		case name == "throughput" && policy.MinimumThroughput > 0 && signal.Value < policy.MinimumThroughput:
			breaches = append(breaches, fmt.Sprintf("throughput %0.3f fell below minimum %0.3f", signal.Value, policy.MinimumThroughput))
		case (name == "unhealthy_instances" || name == "unavailable_replicas") && policy.MaxUnhealthyInstances >= 0 && signal.Value > float64(policy.MaxUnhealthyInstances):
			breaches = append(breaches, fmt.Sprintf("unhealthy instances %0.0f exceeded max %d", signal.Value, policy.MaxUnhealthyInstances))
		case (name == "restart_rate" || name == "crash_rate") && policy.MaxRestartRate > 0 && signal.Value > policy.MaxRestartRate:
			breaches = append(breaches, fmt.Sprintf("restart rate %0.3f exceeded max %0.3f", signal.Value, policy.MaxRestartRate))
		}
	}
	return breaches
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

package app

import (
	"context"
	"fmt"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) applyRiskIntelligence(ctx context.Context, change types.ChangeSet, service types.Service, environment types.Environment, assessment *types.RiskAssessment) {
	if a.Intelligence == nil || !a.Intelligence.Enabled() {
		return
	}

	augmentation, err := a.Intelligence.AugmentRisk(ctx, change, service, environment, *assessment)
	if err != nil {
		assessment.Metadata = withMetadata(assessment.Metadata, "python_intelligence", types.Metadata{
			"status":  "error",
			"source":  "python-subprocess-v1",
			"message": "supplemental intelligence unavailable; deterministic baseline retained",
		})
		assessment.Explanation = append(assessment.Explanation, "supplemental python intelligence was unavailable; deterministic baseline retained")
		return
	}

	assessment.ConfidenceScore = clampFloat(assessment.ConfidenceScore+augmentation.ConfidenceAdjustment, 0.2, 0.99)
	assessment.Explanation = uniqueStrings(append(assessment.Explanation, prefixStrings("python: ", augmentation.SupplementalExplanations)...))
	assessment.RecommendedGuardrails = uniqueStrings(append(assessment.RecommendedGuardrails, augmentation.RecommendedGuardrails...))
	assessment.Metadata = withMetadata(assessment.Metadata, "python_intelligence", types.Metadata{
		"status":                "applied",
		"source":                "python-subprocess-v1",
		"change_cluster":        augmentation.ChangeCluster,
		"confidence_adjustment": augmentation.ConfidenceAdjustment,
		"normalized_factors":    augmentation.NormalizedFactors,
		"historical_pattern":    augmentation.HistoricalPattern,
	})
}

func (a *Application) applyRolloutSimulation(ctx context.Context, change types.ChangeSet, service types.Service, environment types.Environment, assessment types.RiskAssessment, plan *types.RolloutPlan) {
	if a.Intelligence == nil || !a.Intelligence.Enabled() {
		return
	}

	simulation, err := a.Intelligence.SimulateRollout(ctx, change, service, environment, assessment, *plan)
	if err != nil {
		plan.Metadata = withMetadata(plan.Metadata, "python_simulation", types.Metadata{
			"status":  "error",
			"source":  "python-subprocess-v1",
			"message": "supplemental rollout simulation unavailable; deterministic baseline retained",
		})
		plan.Explanation = append(plan.Explanation, "supplemental python rollout simulation was unavailable; deterministic baseline retained")
		return
	}

	plan.VerificationSignals = uniqueStrings(append(plan.VerificationSignals, simulation.VerificationFocus...))
	explanations := append(plan.Explanation, fmt.Sprintf("python simulation recommends %s", simulation.RecommendedNextAction))
	explanations = append(explanations, prefixStrings("python simulation: ", simulation.TimelineNotes)...)
	explanations = append(explanations, prefixStrings("python hotspot: ", simulation.RiskHotspots)...)
	plan.Explanation = uniqueStrings(explanations)
	plan.Metadata = withMetadata(plan.Metadata, "python_simulation", types.Metadata{
		"status":                  "applied",
		"source":                  "python-subprocess-v1",
		"recommended_next_action": simulation.RecommendedNextAction,
		"verification_focus":      simulation.VerificationFocus,
		"risk_hotspots":           simulation.RiskHotspots,
		"timeline_notes":          simulation.TimelineNotes,
		"metadata":                simulation.Metadata,
	})
}

func withMetadata(metadata types.Metadata, key string, value any) types.Metadata {
	if metadata == nil {
		metadata = types.Metadata{}
	}
	metadata[key] = value
	return metadata
}

func prefixStrings(prefix string, items []string) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		result = append(result, prefix+item)
	}
	return result
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func clampFloat(value, lower, upper float64) float64 {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}

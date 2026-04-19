package policies

import (
	"fmt"
	"slices"
	"strings"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

const (
	AppliesToRiskAssessment = "risk_assessment"
	AppliesToRolloutPlan    = "rollout_plan"

	ModeAdvisory            = "advisory"
	ModeBlock               = "block"
	ModeRequireManualReview = "require_manual_review"
)

var (
	allowedAppliesTo = []string{AppliesToRiskAssessment, AppliesToRolloutPlan}
	allowedModes     = []string{ModeAdvisory, ModeBlock, ModeRequireManualReview}
	allowedTouches   = []string{"infrastructure", "secrets", "schema", "dependencies", "poor_rollback_history"}
	allowedMissing   = []string{"observability", "slo"}
)

type EvaluationInput struct {
	Change      types.ChangeSet
	Service     types.Service
	Environment types.Environment
	Assessment  types.RiskAssessment
	AppliesTo   string
}

func NormalizeAppliesTo(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return AppliesToRiskAssessment
	}
	return value
}

func NormalizeMode(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ModeAdvisory
	}
	return value
}

func DetermineScope(projectID, serviceID, environmentID string) string {
	switch {
	case strings.TrimSpace(environmentID) != "":
		return "environment"
	case strings.TrimSpace(serviceID) != "":
		return "service"
	case strings.TrimSpace(projectID) != "":
		return "project"
	default:
		return "organization"
	}
}

func ValidateAppliesTo(value string) error {
	if !slices.Contains(allowedAppliesTo, NormalizeAppliesTo(value)) {
		return fmt.Errorf("unsupported applies_to %q", value)
	}
	return nil
}

func ValidateMode(value string) error {
	if !slices.Contains(allowedModes, NormalizeMode(value)) {
		return fmt.Errorf("unsupported mode %q", value)
	}
	return nil
}

func ValidateConditions(condition types.PolicyCondition) error {
	if strings.TrimSpace(condition.MinRiskLevel) != "" && RiskLevelRank(condition.MinRiskLevel) < 0 {
		return fmt.Errorf("unsupported min_risk_level %q", condition.MinRiskLevel)
	}
	for _, touch := range condition.RequiredTouches {
		if !slices.Contains(allowedTouches, strings.TrimSpace(strings.ToLower(touch))) {
			return fmt.Errorf("unsupported required_touch %q", touch)
		}
	}
	for _, capability := range condition.MissingCapabilities {
		if !slices.Contains(allowedMissing, strings.TrimSpace(strings.ToLower(capability))) {
			return fmt.Errorf("unsupported missing_capability %q", capability)
		}
	}
	return nil
}

func NormalizeConditions(condition types.PolicyCondition) types.PolicyCondition {
	condition.MinRiskLevel = strings.TrimSpace(strings.ToLower(condition.MinRiskLevel))
	condition.RequiredChangeTypes = normalizeUnique(condition.RequiredChangeTypes)
	condition.RequiredTouches = normalizeUnique(condition.RequiredTouches)
	condition.MissingCapabilities = normalizeUnique(condition.MissingCapabilities)
	return condition
}

func ComputeTriggers(condition types.PolicyCondition) []string {
	triggers := make([]string, 0, 6)
	if condition.MinRiskLevel != "" {
		triggers = append(triggers, "risk>="+condition.MinRiskLevel)
	}
	if condition.ProductionOnly {
		triggers = append(triggers, "environment=production")
	}
	if condition.RegulatedOnly {
		triggers = append(triggers, "regulated=true")
	}
	if len(condition.RequiredChangeTypes) > 0 {
		triggers = append(triggers, "change_types="+strings.Join(condition.RequiredChangeTypes, "|"))
	}
	if len(condition.RequiredTouches) > 0 {
		triggers = append(triggers, "touches="+strings.Join(condition.RequiredTouches, "|"))
	}
	if len(condition.MissingCapabilities) > 0 {
		triggers = append(triggers, "missing="+strings.Join(condition.MissingCapabilities, "|"))
	}
	return triggers
}

func EvaluatePolicy(policy types.Policy, input EvaluationInput) ([]string, bool) {
	if !policy.Enabled {
		return nil, false
	}
	if NormalizeAppliesTo(policy.AppliesTo) != NormalizeAppliesTo(input.AppliesTo) {
		return nil, false
	}

	condition := NormalizeConditions(policy.Conditions)
	reasons := make([]string, 0, 6)

	if condition.ProductionOnly {
		if !input.Environment.Production {
			return nil, false
		}
		reasons = append(reasons, "environment is production")
	}

	if condition.RegulatedOnly {
		if !input.Service.RegulatedZone && strings.TrimSpace(input.Environment.ComplianceZone) == "" {
			return nil, false
		}
		reasons = append(reasons, "service or environment is regulated")
	}

	if condition.MinRiskLevel != "" {
		if RiskLevelRank(input.Assessment.Level) < RiskLevelRank(condition.MinRiskLevel) {
			return nil, false
		}
		reasons = append(reasons, fmt.Sprintf("risk level %s meets minimum %s", input.Assessment.Level, condition.MinRiskLevel))
	}

	if len(condition.RequiredChangeTypes) > 0 {
		matchedTypes := intersectNormalized(condition.RequiredChangeTypes, input.Change.ChangeTypes)
		if len(matchedTypes) == 0 {
			return nil, false
		}
		reasons = append(reasons, "change types include "+strings.Join(matchedTypes, ", "))
	}

	if len(condition.RequiredTouches) > 0 {
		matchedTouches := matchedRequiredTouches(condition.RequiredTouches, input.Change)
		if len(matchedTouches) == 0 {
			return nil, false
		}
		reasons = append(reasons, "change touches "+strings.Join(matchedTouches, ", "))
	}

	if len(condition.MissingCapabilities) > 0 {
		missing := matchedMissingCapabilities(condition.MissingCapabilities, input.Service)
		if len(missing) == 0 {
			return nil, false
		}
		reasons = append(reasons, "service lacks "+strings.Join(missing, ", "))
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "scoped policy applies without additional conditions")
	}
	return reasons, true
}

func RiskLevelRank(level any) int {
	var normalized string
	switch typed := level.(type) {
	case types.RiskLevel:
		normalized = string(typed)
	case string:
		normalized = typed
	default:
		return -1
	}
	switch strings.TrimSpace(strings.ToLower(normalized)) {
	case string(types.RiskLevelLow):
		return 0
	case string(types.RiskLevelMedium):
		return 1
	case string(types.RiskLevelHigh):
		return 2
	case string(types.RiskLevelCritical):
		return 3
	default:
		return -1
	}
}

func normalizeUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ToLower(value))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func intersectNormalized(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(left))
	for _, item := range left {
		allowed[strings.TrimSpace(strings.ToLower(item))] = struct{}{}
	}
	matched := make([]string, 0, len(right))
	seen := map[string]struct{}{}
	for _, item := range right {
		normalized := strings.TrimSpace(strings.ToLower(item))
		if normalized == "" {
			continue
		}
		if _, ok := allowed[normalized]; !ok {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		matched = append(matched, normalized)
	}
	return matched
}

func matchedRequiredTouches(required []string, change types.ChangeSet) []string {
	matched := make([]string, 0, len(required))
	for _, touch := range required {
		switch touch {
		case "infrastructure":
			if change.TouchesInfrastructure {
				matched = append(matched, touch)
			}
		case "secrets":
			if change.TouchesSecrets {
				matched = append(matched, touch)
			}
		case "schema":
			if change.TouchesSchema {
				matched = append(matched, touch)
			}
		case "dependencies":
			if change.DependencyChanges {
				matched = append(matched, touch)
			}
		case "poor_rollback_history":
			if change.PoorRollbackHistory {
				matched = append(matched, touch)
			}
		}
	}
	return matched
}

func matchedMissingCapabilities(required []string, service types.Service) []string {
	matched := make([]string, 0, len(required))
	for _, capability := range required {
		switch capability {
		case "observability":
			if !service.HasObservability {
				matched = append(matched, capability)
			}
		case "slo":
			if !service.HasSLO {
				matched = append(matched, capability)
			}
		}
	}
	return matched
}

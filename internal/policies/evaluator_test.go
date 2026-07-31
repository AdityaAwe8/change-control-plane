package policies

import (
	"testing"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestValidateAppliesToCoversGovernedRuntimeSurfaces(t *testing.T) {
	for _, appliesTo := range []string{
		AppliesToRiskAssessment,
		AppliesToRolloutPlan,
		AppliesToRolloutExecution,
		AppliesToReleaseBundle,
		AppliesToConfigSet,
		AppliesToDatabaseGovernance,
		AppliesToChangeWindow,
	} {
		if err := ValidateAppliesTo(appliesTo); err != nil {
			t.Fatalf("expected applies_to %s to be accepted: %v", appliesTo, err)
		}
	}
	if err := ValidateAppliesTo("external_policy_backend"); err == nil {
		t.Fatal("expected speculative external policy backend scope to be rejected")
	}
}

func TestEvaluatePolicyMatchesNewSurfacesAndSkipsDisabledPolicies(t *testing.T) {
	policy := types.Policy{
		AppliesTo: AppliesToReleaseBundle,
		Mode:      ModeBlock,
		Enabled:   true,
		Conditions: types.PolicyCondition{
			ProductionOnly:  true,
			RequiredTouches: []string{"schema"},
		},
	}
	input := EvaluationInput{
		AppliesTo:   AppliesToReleaseBundle,
		Environment: types.Environment{Production: true},
		Change:      types.ChangeSet{TouchesSchema: true},
	}
	reasons, matched := EvaluatePolicy(policy, input)
	if !matched || len(reasons) == 0 {
		t.Fatalf("expected release-bundle policy to match with reasons, matched=%t reasons=%v", matched, reasons)
	}

	policy.Enabled = false
	if _, matched := EvaluatePolicy(policy, input); matched {
		t.Fatal("expected disabled policy not to evaluate")
	}
}

package delivery

import (
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestSimulatedProviderMovesToVerificationGateAfterSignals(t *testing.T) {
	now := time.Now().UTC()
	provider := NewSimulatedProvider()
	result, err := provider.Sync(t.Context(), types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_123", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		SignalSnapshots: []types.SignalSnapshot{
			{
				BaseRecord: types.BaseRecord{ID: "signal_123", CreatedAt: now, UpdatedAt: now},
				Health:     "healthy",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.BackendStatus != "awaiting_verification" {
		t.Fatalf("expected awaiting_verification, got %s", result.BackendStatus)
	}
	if result.ProgressPercent < 80 {
		t.Fatalf("expected progress to reach verification gate, got %d", result.ProgressPercent)
	}
}

func TestNormalizeKubernetesDeploymentStatus(t *testing.T) {
	result := NormalizeKubernetesDeploymentStatus(KubernetesDeploymentStatus{
		Namespace:         "prod",
		DeploymentName:    "checkout",
		Replicas:          4,
		UpdatedReplicas:   4,
		AvailableReplicas: 4,
	})
	if result.BackendStatus != "awaiting_verification" {
		t.Fatalf("expected awaiting_verification status, got %s", result.BackendStatus)
	}
	if result.ProgressPercent != 100 {
		t.Fatalf("expected 100 progress, got %d", result.ProgressPercent)
	}
}

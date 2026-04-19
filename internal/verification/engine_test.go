package verification

import (
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/delivery"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestEngineRollsBackCriticalProductionSignals(t *testing.T) {
	engine := NewEngine()
	now := time.Now().UTC()
	evaluation := engine.Evaluate(types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_123", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		Assessment: types.RiskAssessment{Level: types.RiskLevelHigh},
		Service:    types.Service{Criticality: "mission_critical", CustomerFacing: true},
		Environment: types.Environment{
			Production: true,
		},
		SignalSnapshots: []types.SignalSnapshot{
			{
				BaseRecord: types.BaseRecord{ID: "signal_123", CreatedAt: now, UpdatedAt: now},
				Health:     "critical",
				Summary:    "latency and error budgets breached",
				Signals: []types.SignalValue{
					{Name: "latency", Category: "technical", Status: "critical"},
				},
			},
		},
		Plan: types.RolloutPlan{
			VerificationSignals: []string{"latency"},
		},
	}, delivery.SyncResult{
		BackendType:   "simulated",
		BackendStatus: "awaiting_verification",
		Summary:       "awaiting signal gate",
	})
	if !evaluation.Record {
		t.Fatal("expected automated verification decision")
	}
	if evaluation.Request.Decision != "rollback" {
		t.Fatalf("expected rollback, got %s", evaluation.Request.Decision)
	}
}

func TestNormalizePrometheusSignal(t *testing.T) {
	value := NormalizePrometheusSignal("latency_p95", "technical", 420, 250, ">")
	if value.Status != "degraded" {
		t.Fatalf("expected degraded status, got %s", value.Status)
	}
}

func TestEngineRollsBackWhenPolicyThresholdsAreBreached(t *testing.T) {
	engine := NewEngine()
	now := time.Now().UTC()
	evaluation := engine.Evaluate(types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_456", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		EffectiveRollbackPolicy: &types.RollbackPolicy{
			BaseRecord:               types.BaseRecord{ID: "policy_prod"},
			Name:                     "Production strict",
			MaxErrorRate:             1,
			RollbackOnCriticalSignals: true,
		},
		SignalSnapshots: []types.SignalSnapshot{
			{
				BaseRecord: types.BaseRecord{ID: "signal_456", CreatedAt: now, UpdatedAt: now},
				Health:     "healthy",
				Summary:    "provider health looked healthy",
				Signals: []types.SignalValue{
					{Name: "error_rate", Category: "technical", Value: 2.4, Status: "healthy"},
				},
			},
		},
		Plan: types.RolloutPlan{
			VerificationSignals: []string{"error_rate"},
		},
	}, delivery.SyncResult{
		BackendType:   "prometheus",
		BackendStatus: "awaiting_verification",
		Summary:       "signal gate open",
	})
	if !evaluation.Record {
		t.Fatal("expected automated evaluation")
	}
	if evaluation.Request.Decision != "rollback" {
		t.Fatalf("expected rollback decision, got %s", evaluation.Request.Decision)
	}
	if evaluation.Request.Metadata["policy_name"] != "Production strict" {
		t.Fatalf("expected policy metadata to be recorded, got %+v", evaluation.Request.Metadata)
	}
}

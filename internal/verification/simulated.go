package verification

import (
	"context"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type SimulatedProvider struct{}

func NewSimulatedProvider() SignalProvider {
	return SimulatedProvider{}
}

func (SimulatedProvider) Kind() string {
	return "simulated"
}

func (SimulatedProvider) Collect(_ context.Context, runtime types.RolloutExecutionRuntimeContext) (Collection, error) {
	return Collection{
		Source:      "persisted-simulated-snapshots",
		Explanation: []string{"simulated signal provider reads normalized snapshots that were ingested through the control-plane API"},
		CollectedAt: time.Now().UTC(),
	}, nil
}

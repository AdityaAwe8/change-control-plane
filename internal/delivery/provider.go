package delivery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var ErrProviderUnavailable = errors.New("orchestrator provider unavailable")

type ProviderError struct {
	Operation  string
	StatusCode int
	Temporary  bool
	Err        error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s failed with status %d: %v", e.Operation, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("%s failed: %v", e.Operation, e.Err)
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *ProviderError) Retryable() bool {
	if e == nil {
		return false
	}
	return e.Temporary
}

type SyncResult struct {
	BackendType        string         `json:"backend_type"`
	BackendExecutionID string         `json:"backend_execution_id,omitempty"`
	BackendStatus      string         `json:"backend_status"`
	ProgressPercent    int            `json:"progress_percent"`
	CurrentStep        string         `json:"current_step,omitempty"`
	Summary            string         `json:"summary,omitempty"`
	Explanation        []string       `json:"explanation,omitempty"`
	Metadata           types.Metadata `json:"metadata,omitempty"`
	LastUpdatedAt      time.Time      `json:"last_updated_at"`
}

func (s SyncResult) Terminal() bool {
	switch s.BackendStatus {
	case "succeeded", "failed", "rolled_back":
		return true
	default:
		return false
	}
}

type Provider interface {
	Kind() string
	Submit(context.Context, types.RolloutExecutionRuntimeContext) (SyncResult, error)
	Sync(context.Context, types.RolloutExecutionRuntimeContext) (SyncResult, error)
	Pause(context.Context, types.RolloutExecutionRuntimeContext, string) (SyncResult, error)
	Resume(context.Context, types.RolloutExecutionRuntimeContext, string) (SyncResult, error)
	Rollback(context.Context, types.RolloutExecutionRuntimeContext, string) (SyncResult, error)
}

type Registry struct {
	providers   map[string]Provider
	defaultKind string
}

func NewRegistry() *Registry {
	registry := &Registry{
		providers:   map[string]Provider{},
		defaultKind: "simulated",
	}
	registry.Register(NewSimulatedProvider())
	registry.Register(NewKubernetesDeploymentProvider())
	return registry
}

func (r *Registry) Register(provider Provider) {
	if provider == nil {
		return
	}
	kind := strings.TrimSpace(strings.ToLower(provider.Kind()))
	if kind == "" {
		return
	}
	r.providers[kind] = provider
}

func (r *Registry) Resolve(kind string) (Provider, error) {
	resolved := strings.TrimSpace(strings.ToLower(kind))
	if resolved == "" {
		resolved = r.defaultKind
	}
	provider, ok := r.providers[resolved]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnavailable, resolved)
	}
	return provider, nil
}

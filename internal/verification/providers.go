package verification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var ErrSignalProviderUnavailable = errors.New("signal provider unavailable")

type SignalProviderError struct {
	Operation  string
	StatusCode int
	Temporary  bool
	Err        error
}

func (e *SignalProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s failed with status %d: %v", e.Operation, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("%s failed: %v", e.Operation, e.Err)
}

func (e *SignalProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *SignalProviderError) Retryable() bool {
	if e == nil {
		return false
	}
	return e.Temporary
}

type Collection struct {
	Snapshots   []types.SignalSnapshot `json:"snapshots,omitempty"`
	Source      string                 `json:"source"`
	Explanation []string               `json:"explanation,omitempty"`
	CollectedAt time.Time              `json:"collected_at"`
}

type SignalProvider interface {
	Kind() string
	Collect(context.Context, types.RolloutExecutionRuntimeContext) (Collection, error)
}

type Registry struct {
	providers   map[string]SignalProvider
	defaultKind string
}

func NewRegistry() *Registry {
	registry := &Registry{
		providers:   map[string]SignalProvider{},
		defaultKind: "simulated",
	}
	registry.Register(NewSimulatedProvider())
	registry.Register(NewPrometheusProvider())
	return registry
}

func (r *Registry) Register(provider SignalProvider) {
	if provider == nil {
		return
	}
	kind := strings.TrimSpace(strings.ToLower(provider.Kind()))
	if kind == "" {
		return
	}
	r.providers[kind] = provider
}

func (r *Registry) Resolve(kind string) (SignalProvider, error) {
	resolved := strings.TrimSpace(strings.ToLower(kind))
	if resolved == "" {
		resolved = r.defaultKind
	}
	provider, ok := r.providers[resolved]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSignalProviderUnavailable, resolved)
	}
	return provider, nil
}

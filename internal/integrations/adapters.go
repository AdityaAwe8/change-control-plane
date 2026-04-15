package integrations

import (
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Adapter interface {
	Descriptor() types.Integration
}

type StaticAdapter struct {
	descriptor types.Integration
}

func NewStaticAdapter(name, kind, description string, capabilities []string) StaticAdapter {
	now := time.Now().UTC()
	return StaticAdapter{
		descriptor: types.Integration{
			BaseRecord: types.BaseRecord{
				ID:        "integration_" + kind,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:         name,
			Kind:         kind,
			Mode:         "advisory-ready",
			Status:       "available",
			Capabilities: capabilities,
			Description:  description,
		},
	}
}

func (a StaticAdapter) Descriptor() types.Integration {
	return a.descriptor
}

type Registry struct {
	adapters []Adapter
}

func NewRegistry() *Registry {
	return &Registry{
		adapters: []Adapter{
			NewStaticAdapter("GitHub", "github", "Repository and change metadata ingestion with workflow governance hooks.", []string{"scm", "pull_requests", "workflow_metadata"}),
			NewStaticAdapter("Kubernetes", "kubernetes", "Cluster and workload topology awareness for rollout safety and environment modeling.", []string{"workloads", "namespaces", "rollout_targets"}),
			NewStaticAdapter("Slack", "slack", "Notification and approval workflow surface for operational collaboration.", []string{"notifications", "approvals", "incident_channels"}),
			NewStaticAdapter("Jira", "jira", "Change traceability, ticket correlation, and evidence linking.", []string{"tickets", "change_context", "compliance_traceability"}),
		},
	}
}

func (r *Registry) List() []types.Integration {
	items := make([]types.Integration, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		items = append(items, adapter.Descriptor())
	}
	return items
}

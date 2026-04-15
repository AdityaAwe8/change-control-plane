package audit

import (
	"context"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Actor struct {
	ID    string
	Type  string
	Label string
}

type Recorder interface {
	Record(context.Context, Actor, string, string, string, string, string, string, []string) (types.AuditEvent, error)
}

type Sink interface {
	CreateAuditEvent(context.Context, types.AuditEvent) error
}

type Service struct {
	sink Sink
}

func NewService(sink Sink) *Service {
	return &Service{sink: sink}
}

func (s *Service) Record(ctx context.Context, actor Actor, action, resourceType, resourceID, outcome, organizationID, projectID string, details []string) (types.AuditEvent, error) {
	now := time.Now().UTC()
	event := types.AuditEvent{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("audit"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: organizationID,
		ProjectID:      projectID,
		ActorID:        actor.ID,
		ActorType:      actor.Type,
		Actor:          actor.Label,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		Outcome:        outcome,
		Details:        details,
	}
	if err := s.sink.CreateAuditEvent(ctx, event); err != nil {
		return types.AuditEvent{}, err
	}
	return event, nil
}

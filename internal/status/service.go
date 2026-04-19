package status

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

type RecordRequest struct {
	OrganizationID     string
	ProjectID          string
	TeamID             string
	ServiceID          string
	EnvironmentID      string
	RolloutExecutionID string
	ChangeSetID        string
	ResourceType       string
	ResourceID         string
	EventType          string
	Category           string
	Severity           string
	PreviousState      string
	NewState           string
	Outcome            string
	Source             string
	Automated          bool
	Summary            string
	Explanation        []string
	CorrelationID      string
	Metadata           types.Metadata
}

type Sink interface {
	CreateStatusEvent(context.Context, types.StatusEvent) error
}

type Service struct {
	sink Sink
}

func NewService(sink Sink) *Service {
	return &Service{sink: sink}
}

func (s *Service) Record(ctx context.Context, actor Actor, req RecordRequest) (types.StatusEvent, error) {
	now := time.Now().UTC()
	event := types.StatusEvent{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("status"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:     req.OrganizationID,
		ProjectID:          req.ProjectID,
		TeamID:             req.TeamID,
		ServiceID:          req.ServiceID,
		EnvironmentID:      req.EnvironmentID,
		RolloutExecutionID: req.RolloutExecutionID,
		ChangeSetID:        req.ChangeSetID,
		ResourceType:       req.ResourceType,
		ResourceID:         req.ResourceID,
		EventType:          req.EventType,
		Category:           req.Category,
		Severity:           req.Severity,
		PreviousState:      req.PreviousState,
		NewState:           req.NewState,
		Outcome:            req.Outcome,
		ActorID:            actor.ID,
		ActorType:          actor.Type,
		Actor:              actor.Label,
		Source:             req.Source,
		Automated:          req.Automated,
		Summary:            req.Summary,
		Explanation:        req.Explanation,
		CorrelationID:      req.CorrelationID,
	}
	if err := s.sink.CreateStatusEvent(ctx, event); err != nil {
		return types.StatusEvent{}, err
	}
	return event, nil
}

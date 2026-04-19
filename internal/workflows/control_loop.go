package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type ControlLoop struct {
	app         *app.Application
	autoAdvance bool
}

type Summary struct {
	Scanned             int
	Claimed             int
	Started             int
	Completed           int
	AutomatedDecisions  int
	IntegrationsScanned int
	SyncsClaimed        int
	SyncsCompleted      int
	EventsDispatched    int
	Failures            []string
}

func NewControlLoop(application *app.Application, autoAdvance bool) *ControlLoop {
	return &ControlLoop{
		app:         application,
		autoAdvance: autoAdvance,
	}
}

func (c *ControlLoop) RunOnce(ctx context.Context) (Summary, error) {
	summary := Summary{}
	dispatched, err := c.app.Events.DispatchPending(ctx, 50)
	if err != nil {
		summary.Failures = append(summary.Failures, fmt.Sprintf("outbox:dispatch:%s", err.Error()))
	} else {
		summary.EventsDispatched = dispatched
	}
	executions, err := c.app.ListRolloutExecutions(ctx)
	if err != nil {
		return summary, err
	}

	for _, execution := range executions {
		summary.Scanned++
		claimedAt := time.Now().UTC()
		claimed, err := c.app.ClaimRolloutExecution(ctx, execution.ID, claimedAt, claimedAt)
		if err != nil {
			summary.Failures = append(summary.Failures, fmt.Sprintf("%s:claim:%s", execution.ID, err.Error()))
			continue
		}
		if !claimed {
			continue
		}
		summary.Claimed++
		action, reason, ok := c.nextAction(execution)
		if !ok {
		} else {
			updated, err := c.app.AdvanceRolloutExecution(ctx, execution.ID, types.AdvanceRolloutExecutionRequest{
				Action: action,
				Reason: reason,
			})
			if err != nil {
				summary.Failures = append(summary.Failures, fmt.Sprintf("%s:%s", execution.ID, err.Error()))
				continue
			}
			execution = updated
			switch action {
			case "start":
				summary.Started++
			case "complete":
				summary.Completed++
			}
		}
		detail, err := c.app.ReconcileRolloutExecution(ctx, execution.ID)
		if err != nil {
			summary.Failures = append(summary.Failures, fmt.Sprintf("%s:reconcile:%s", execution.ID, err.Error()))
			continue
		}
		if len(detail.VerificationResults) > 0 &&
			detail.VerificationResults[len(detail.VerificationResults)-1].Automated &&
			detail.VerificationResults[len(detail.VerificationResults)-1].ID != execution.LastVerificationResult {
			summary.AutomatedDecisions++
		}
	}

	integrations, err := c.app.IntegrationsList(ctx)
	if err != nil {
		return summary, err
	}
	for _, integration := range integrations {
		summary.IntegrationsScanned++
		if !integration.Enabled || !integration.ScheduleEnabled || integration.NextScheduledSyncAt == nil {
			continue
		}
		now := time.Now().UTC()
		if integration.NextScheduledSyncAt.After(now) {
			continue
		}
		claimed, err := c.app.ClaimScheduledIntegrationSync(ctx, integration.ID, now)
		if err != nil {
			summary.Failures = append(summary.Failures, fmt.Sprintf("%s:claim-sync:%s", integration.ID, err.Error()))
			continue
		}
		if !claimed {
			continue
		}
		summary.SyncsClaimed++
		if _, err := c.app.RunScheduledIntegrationSync(ctx, integration.ID); err != nil {
			summary.Failures = append(summary.Failures, fmt.Sprintf("%s:scheduled-sync:%s", integration.ID, err.Error()))
			continue
		}
		summary.SyncsCompleted++
	}
	return summary, nil
}

func (c *ControlLoop) nextAction(execution types.RolloutExecution) (string, string, bool) {
	if !c.autoAdvance {
		return "", "", false
	}
	switch execution.Status {
	case "planned", "approved":
		return "start", "worker control loop auto-started rollout execution", true
	case "verified":
		return "complete", "worker control loop marked verified rollout execution complete", true
	default:
		return "", "", false
	}
}

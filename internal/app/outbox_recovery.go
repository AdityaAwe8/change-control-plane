package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) RetryOutboxEvent(ctx context.Context, eventID string) (types.OutboxEvent, error) {
	return a.recoverOutboxEvent(ctx, eventID, "error", "retry", "outbox_event.retry", "outbox_event.retry.denied")
}

func (a *Application) RequeueOutboxEvent(ctx context.Context, eventID string) (types.OutboxEvent, error) {
	return a.recoverOutboxEvent(ctx, eventID, "dead_letter", "requeue", "outbox_event.requeue", "outbox_event.requeue.denied")
}

func (a *Application) recoverOutboxEvent(ctx context.Context, eventID, requiredStatus, action, eventType, deniedEventType string) (types.OutboxEvent, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.OutboxEvent{}, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return types.OutboxEvent{}, err
	}
	if !a.Authorizer.CanManageOrganization(identity, orgID) {
		return types.OutboxEvent{}, a.forbidden(ctx, identity, deniedEventType, "outbox_event", eventID, orgID, "", []string{"actor lacks durable outbox recovery permission"})
	}
	if strings.TrimSpace(eventID) == "" {
		return types.OutboxEvent{}, fmt.Errorf("%w: outbox event id is required", ErrValidation)
	}

	item, err := a.Store.GetOutboxEvent(ctx, eventID)
	if err != nil {
		return types.OutboxEvent{}, err
	}
	if item.OrganizationID != orgID {
		return types.OutboxEvent{}, a.forbidden(ctx, identity, deniedEventType, "outbox_event", item.ID, orgID, "", []string{"outbox event does not belong to the active organization"})
	}
	if item.Status != requiredStatus {
		return types.OutboxEvent{}, invalidOutboxRecoveryStatus(requiredStatus, action, item.Status)
	}

	previousStatus := item.Status
	now := time.Now().UTC()
	item.Status = "pending"
	item.ClaimedAt = nil
	item.NextAttemptAt = nil
	item.ProcessedAt = nil
	item.UpdatedAt = now
	item.Metadata = appendOutboxRecoveryMetadata(item.Metadata, identity, action, previousStatus, item)

	err = a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		ok, err := a.Store.UpdateOutboxEventIfStatus(txCtx, item, previousStatus)
		if err != nil {
			return err
		}
		if !ok {
			current, currentErr := a.Store.GetOutboxEvent(txCtx, item.ID)
			if currentErr != nil {
				if currentErr == storage.ErrNotFound {
					return currentErr
				}
				return currentErr
			}
			return invalidOutboxRecoveryStatus(requiredStatus, action, current.Status)
		}
		return a.recordOutboxRecoveryAction(txCtx, identity, eventType, action, previousStatus, item)
	})
	if err != nil {
		return types.OutboxEvent{}, err
	}
	return item, nil
}

func (a *Application) recordOutboxRecoveryAction(ctx context.Context, identity auth.Identity, eventType, action, previousStatus string, item types.OutboxEvent) error {
	details := []string{
		fmt.Sprintf("manual %s requested", action),
		fmt.Sprintf("previous_status=%s", previousStatus),
		fmt.Sprintf("attempts=%d", item.Attempts),
	}
	if item.LastError != "" {
		details = append(details, fmt.Sprintf("last_error=%s", item.LastError))
	}
	auditEvent, err := a.Audit.Record(ctx, auditActorFromIdentity(identity), eventType, "outbox_event", item.ID, "success", item.OrganizationID, item.ProjectID, details)
	if err != nil {
		return err
	}
	return a.emitStatusEventFromAudit(
		ctx,
		identity,
		auditEvent,
		details,
		withStatusCategory("operations"),
		withStatusSource("api"),
		withStatusSeverity("warning"),
		withStatusStates(previousStatus, item.Status),
		withStatusSummary(fmt.Sprintf("outbox event moved from %s to pending via manual %s", previousStatus, action)),
		withStatusMetadata(types.Metadata{
			"manual_recovery_action":       action,
			"manual_recovery_requested_at": item.UpdatedAt.Format(time.RFC3339Nano),
			"last_error_class":             stringMetadataValue(item.Metadata, "last_error_class"),
			"recovery_hint":                stringMetadataValue(item.Metadata, "recovery_hint"),
		}),
	)
}

func appendOutboxRecoveryMetadata(metadata types.Metadata, identity auth.Identity, action, previousStatus string, item types.OutboxEvent) types.Metadata {
	metadata = ensureMetadataMap(metadata)
	entry := types.Metadata{
		"action":          action,
		"requested_at":    item.UpdatedAt.Format(time.RFC3339Nano),
		"actor_id":        identity.ActorID,
		"actor_type":      string(identity.ActorType),
		"actor":           identity.ActorLabel(),
		"previous_status": previousStatus,
		"next_status":     item.Status,
		"attempts":        item.Attempts,
		"last_error":      item.LastError,
	}
	if errorClass := stringMetadataValue(item.Metadata, "last_error_class"); errorClass != "" {
		entry["last_error_class"] = errorClass
	}
	if deadLetteredAt := stringMetadataValue(item.Metadata, "dead_lettered_at"); deadLetteredAt != "" {
		entry["dead_lettered_at"] = deadLetteredAt
	}
	if recoveryHint := stringMetadataValue(item.Metadata, "recovery_hint"); recoveryHint != "" {
		entry["recovery_hint"] = recoveryHint
	}
	metadata["manual_recovery_history"] = appendMetadataEntry(metadata["manual_recovery_history"], entry)
	metadata["manual_recovery_last_action"] = action
	metadata["manual_recovery_last_actor_id"] = identity.ActorID
	metadata["manual_recovery_last_actor_type"] = string(identity.ActorType)
	metadata["manual_recovery_last_actor"] = identity.ActorLabel()
	metadata["manual_recovery_last_requested_at"] = item.UpdatedAt.Format(time.RFC3339Nano)
	return metadata
}

func appendMetadataEntry(raw any, entry types.Metadata) []any {
	items := make([]any, 0, 4)
	switch typed := raw.(type) {
	case []any:
		items = append(items, typed...)
	case []types.Metadata:
		for _, item := range typed {
			items = append(items, item)
		}
	}
	items = append(items, entry)
	if len(items) <= 10 {
		return items
	}
	return append([]any(nil), items[len(items)-10:]...)
}

func ensureMetadataMap(metadata types.Metadata) types.Metadata {
	if metadata == nil {
		return types.Metadata{}
	}
	return metadata
}

func actionVerb(action string) string {
	switch strings.TrimSpace(action) {
	case "retry":
		return "retried"
	case "requeue":
		return "requeued"
	default:
		return action
	}
}

func invalidOutboxRecoveryStatus(requiredStatus, action, currentStatus string) error {
	return fmt.Errorf("%w: manual %s only applies to %s outbox events; current status is %s", ErrValidation, action, requiredStatus, valueOrDefault(strings.TrimSpace(currentStatus), "unknown"))
}

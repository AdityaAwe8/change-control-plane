package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Handler func(context.Context, types.DomainEvent) error

type Bus interface {
	Publish(context.Context, types.DomainEvent) error
	Subscribe(string, Handler)
	DispatchPending(context.Context, int) (int, error)
}

type InMemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
}

func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{
		subscribers: make(map[string][]Handler),
	}
}

func (b *InMemoryBus) Publish(ctx context.Context, event types.DomainEvent) error {
	return b.dispatch(ctx, event)
}

func (b *InMemoryBus) dispatch(ctx context.Context, event types.DomainEvent) error {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.subscribers[event.Type]...)
	wildcards := append([]Handler(nil), b.subscribers["*"]...)
	b.mu.RUnlock()

	for _, handler := range append(handlers, wildcards...) {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (b *InMemoryBus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *InMemoryBus) DispatchPending(_ context.Context, _ int) (int, error) {
	return 0, nil
}

type OutboxStore interface {
	CreateOutboxEvent(context.Context, types.OutboxEvent) error
	ClaimOutboxEvents(context.Context, time.Time, int, time.Time) ([]types.OutboxEvent, error)
	UpdateOutboxEvent(context.Context, types.OutboxEvent) error
}

type DurableBus struct {
	store      OutboxStore
	inMemory   *InMemoryBus
	retryDelay time.Duration
	maxAttempts int
}

func NewDurableBus(store OutboxStore) *DurableBus {
	return &DurableBus{
		store:      store,
		inMemory:   NewInMemoryBus(),
		retryDelay: 30 * time.Second,
		maxAttempts: 5,
	}
}

func (b *DurableBus) Publish(ctx context.Context, event types.DomainEvent) error {
	now := time.Now().UTC()
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now
	}
	outboxEvent := types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        valueOrDefault(event.ID, fmt.Sprintf("evt_%d", now.UnixNano())),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata: types.Metadata{
				"occurred_at":  event.OccurredAt.Format(time.RFC3339Nano),
				"payload":      event.Payload,
				"resource_id":  event.ResourceID,
				"resource_type": event.ResourceType,
			},
		},
		EventType:      event.Type,
		OrganizationID: event.OrganizationID,
		ProjectID:      event.ProjectID,
		ResourceType:   event.ResourceType,
		ResourceID:     event.ResourceID,
		Status:         "pending",
	}
	return b.store.CreateOutboxEvent(ctx, outboxEvent)
}

func (b *DurableBus) Subscribe(eventType string, handler Handler) {
	b.inMemory.Subscribe(eventType, handler)
}

func (b *DurableBus) DispatchPending(ctx context.Context, limit int) (int, error) {
	now := time.Now().UTC()
	items, err := b.store.ClaimOutboxEvents(ctx, now, limit, now.Add(-2*time.Minute))
	if err != nil {
		return 0, err
	}
	dispatched := 0
	for _, item := range items {
		event := domainEventFromOutbox(item)
		dispatchErr := b.inMemory.dispatch(ctx, event)
		item.Attempts++
		item.ClaimedAt = nil
		item.UpdatedAt = time.Now().UTC()
		item.Metadata = ensureMetadata(item.Metadata)
		item.Metadata["last_dispatch_completed_at"] = item.UpdatedAt.Format(time.RFC3339Nano)
		if dispatchErr != nil {
			item.LastError = dispatchErr.Error()
			errorClass := classifyDispatchError(dispatchErr)
			item.Metadata["last_error_class"] = errorClass
			if shouldDeadLetter(item.Attempts, b.maxAttempts, dispatchErr) {
				item.Status = "dead_letter"
				item.NextAttemptAt = nil
				item.Metadata["dead_lettered_at"] = item.UpdatedAt.Format(time.RFC3339Nano)
				item.Metadata["recovery_hint"] = recoveryHintForError(errorClass)
			} else {
				item.Status = "error"
				nextAttemptAt := item.UpdatedAt.Add(b.backoff(item.Attempts))
				item.NextAttemptAt = &nextAttemptAt
			}
			if err := b.store.UpdateOutboxEvent(ctx, item); err != nil {
				return dispatched, err
			}
			continue
		}
		item.Status = "processed"
		item.LastError = ""
		item.NextAttemptAt = nil
		delete(item.Metadata, "last_error_class")
		delete(item.Metadata, "dead_lettered_at")
		delete(item.Metadata, "recovery_hint")
		processedAt := item.UpdatedAt
		item.ProcessedAt = &processedAt
		if err := b.store.UpdateOutboxEvent(ctx, item); err != nil {
			return dispatched, err
		}
		dispatched++
	}
	return dispatched, nil
}

func (b *DurableBus) backoff(attempts int) time.Duration {
	if attempts <= 1 {
		return b.retryDelay
	}
	delay := b.retryDelay * time.Duration(1<<(attempts-1))
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}

type retryableError interface {
	Retryable() bool
}

func shouldDeadLetter(attempts, maxAttempts int, err error) bool {
	if attempts >= maxAttempts {
		return true
	}
	var retryable retryableError
	if errors.As(err, &retryable) {
		return !retryable.Retryable()
	}
	return false
}

func classifyDispatchError(err error) string {
	var retryable retryableError
	if errors.As(err, &retryable) {
		if retryable.Retryable() {
			return "temporary"
		}
		return "permanent"
	}
	return "unclassified"
}

func recoveryHintForError(errorClass string) string {
	switch errorClass {
	case "permanent":
		return "fix the handler or payload before replaying this event"
	case "temporary":
		return "check upstream dependency health before forcing an immediate retry"
	default:
		return "inspect the handler and retry diagnostics before replaying"
	}
}

func domainEventFromOutbox(item types.OutboxEvent) types.DomainEvent {
	event := types.DomainEvent{
		ID:             item.ID,
		Type:           item.EventType,
		OrganizationID: item.OrganizationID,
		ProjectID:      item.ProjectID,
		ResourceType:   item.ResourceType,
		ResourceID:     item.ResourceID,
		OccurredAt:     item.CreatedAt,
	}
	if item.Metadata == nil {
		return event
	}
	if payload, ok := item.Metadata["payload"].(map[string]any); ok {
		event.Payload = payload
	}
	if occurredAt, ok := item.Metadata["occurred_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339Nano, occurredAt); err == nil {
			event.OccurredAt = parsed
		}
	}
	return event
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func ensureMetadata(metadata types.Metadata) types.Metadata {
	if metadata == nil {
		return types.Metadata{}
	}
	return metadata
}

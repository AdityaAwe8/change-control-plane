package events

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestDurableBusPublishAndDispatchPendingMarksProcessed(t *testing.T) {
	store := newOutboxTestStore()
	bus := NewDurableBus(store)

	var received types.DomainEvent
	bus.Subscribe("status.created", func(_ context.Context, event types.DomainEvent) error {
		received = event
		return nil
	})

	err := bus.Publish(context.Background(), types.DomainEvent{
		ID:             "evt_status_1",
		Type:           "status.created",
		OrganizationID: "org_123",
		ResourceType:   "status_event",
		ResourceID:     "status_123",
		Payload:        types.Metadata{"summary": "rollout execution observed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one outbox event, got %d", len(store.events))
	}

	dispatched, err := bus.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched != 1 {
		t.Fatalf("expected one dispatched event, got %d", dispatched)
	}
	if received.Type != "status.created" || received.ResourceID != "status_123" {
		t.Fatalf("unexpected dispatched event %+v", received)
	}

	item := store.mustGet("evt_status_1")
	if item.Status != "processed" || item.ProcessedAt == nil {
		t.Fatalf("expected processed outbox event, got %+v", item)
	}
}

func TestDurableBusRetriesAfterFailure(t *testing.T) {
	store := newOutboxTestStore()
	bus := NewDurableBus(store)

	failOnce := true
	bus.Subscribe("sync.requested", func(_ context.Context, event types.DomainEvent) error {
		if event.ResourceID != "integration_123" {
			t.Fatalf("unexpected resource id %q", event.ResourceID)
		}
		if failOnce {
			failOnce = false
			return temporaryDispatchError{err: errors.New("temporary dispatcher failure")}
		}
		return nil
	})

	if err := bus.Publish(context.Background(), types.DomainEvent{
		ID:             "evt_sync_1",
		Type:           "sync.requested",
		OrganizationID: "org_123",
		ResourceType:   "integration",
		ResourceID:     "integration_123",
	}); err != nil {
		t.Fatal(err)
	}

	dispatched, err := bus.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched != 0 {
		t.Fatalf("expected no successful dispatches on first pass, got %d", dispatched)
	}

	item := store.mustGet("evt_sync_1")
	if item.Status != "error" || item.Attempts != 1 || item.NextAttemptAt == nil {
		t.Fatalf("expected retry state after failure, got %+v", item)
	}

	retryAt := time.Now().UTC().Add(-1 * time.Second)
	item.NextAttemptAt = &retryAt
	item.Status = "pending"
	item.UpdatedAt = retryAt
	if err := store.UpdateOutboxEvent(context.Background(), item); err != nil {
		t.Fatal(err)
	}

	dispatched, err = bus.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched != 1 {
		t.Fatalf("expected one successful retry dispatch, got %d", dispatched)
	}

	retried := store.mustGet("evt_sync_1")
	if retried.Status != "processed" || retried.Attempts != 2 || retried.ProcessedAt == nil {
		t.Fatalf("expected processed retry event, got %+v", retried)
	}
}

func TestDurableBusDeadLettersPermanentFailures(t *testing.T) {
	store := newOutboxTestStore()
	bus := NewDurableBus(store)

	bus.Subscribe("webhook.received", func(_ context.Context, _ types.DomainEvent) error {
		return permanentDispatchError{err: errors.New("malformed payload")}
	})

	if err := bus.Publish(context.Background(), types.DomainEvent{
		ID:             "evt_dead_1",
		Type:           "webhook.received",
		OrganizationID: "org_123",
		ResourceType:   "integration",
		ResourceID:     "github_123",
	}); err != nil {
		t.Fatal(err)
	}

	dispatched, err := bus.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched != 0 {
		t.Fatalf("expected dead-lettered event to count as zero successful dispatches, got %d", dispatched)
	}

	item := store.mustGet("evt_dead_1")
	if item.Status != "dead_letter" {
		t.Fatalf("expected dead_letter status, got %+v", item)
	}
	if item.Metadata["last_error_class"] != "permanent" {
		t.Fatalf("expected permanent error class metadata, got %+v", item.Metadata)
	}
	if _, ok := item.Metadata["dead_lettered_at"]; !ok {
		t.Fatalf("expected dead-letter timestamp metadata, got %+v", item.Metadata)
	}
}

func TestDurableBusReclaimsStaleProcessingEventsAfterRestartWindow(t *testing.T) {
	store := newOutboxTestStore()
	bus := NewDurableBus(store)

	handled := 0
	bus.Subscribe("status.created", func(_ context.Context, _ types.DomainEvent) error {
		handled++
		return nil
	})

	now := time.Now().UTC().Add(-5 * time.Minute)
	store.events["evt_reclaim_1"] = types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        "evt_reclaim_1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		EventType:      "status.created",
		OrganizationID: "org_123",
		ResourceType:   "status_event",
		ResourceID:     "status_123",
		Status:         "processing",
		Attempts:       1,
		ClaimedAt:      &now,
	}

	dispatched, err := bus.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched != 1 || handled != 1 {
		t.Fatalf("expected stale processing event to be reclaimed and dispatched, dispatched=%d handled=%d", dispatched, handled)
	}

	item := store.mustGet("evt_reclaim_1")
	if item.Status != "processed" || item.ProcessedAt == nil {
		t.Fatalf("expected reclaimed event to finish as processed, got %+v", item)
	}
}

func TestDurableBusLeavesFreshProcessingAndDeadLetterEventsUnclaimed(t *testing.T) {
	store := newOutboxTestStore()
	bus := NewDurableBus(store)

	handled := 0
	bus.Subscribe("status.created", func(_ context.Context, event types.DomainEvent) error {
		if event.ID != "evt_pending_dispatchable" {
			t.Fatalf("unexpected event dispatched: %+v", event)
		}
		handled++
		return nil
	})

	now := time.Now().UTC()
	freshClaim := now.Add(-30 * time.Second)
	store.events["evt_pending_dispatchable"] = types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        "evt_pending_dispatchable",
			CreatedAt: now.Add(-5 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
		},
		EventType:      "status.created",
		OrganizationID: "org_123",
		ResourceType:   "status_event",
		ResourceID:     "status_pending",
		Status:         "pending",
	}
	store.events["evt_processing_fresh"] = types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        "evt_processing_fresh",
			CreatedAt: now.Add(-4 * time.Minute),
			UpdatedAt: now.Add(-30 * time.Second),
		},
		EventType:      "status.created",
		OrganizationID: "org_123",
		ResourceType:   "status_event",
		ResourceID:     "status_processing",
		Status:         "processing",
		Attempts:       1,
		ClaimedAt:      &freshClaim,
	}
	store.events["evt_dead_letter_blocked"] = types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        "evt_dead_letter_blocked",
			CreatedAt: now.Add(-3 * time.Minute),
			UpdatedAt: now.Add(-2 * time.Minute),
		},
		EventType:      "status.created",
		OrganizationID: "org_123",
		ResourceType:   "status_event",
		ResourceID:     "status_dead_letter",
		Status:         "dead_letter",
		Attempts:       5,
	}

	dispatched, err := bus.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched != 1 || handled != 1 {
		t.Fatalf("expected only the pending event to dispatch, dispatched=%d handled=%d", dispatched, handled)
	}

	pending := store.mustGet("evt_pending_dispatchable")
	if pending.Status != "processed" || pending.ProcessedAt == nil {
		t.Fatalf("expected pending event to finish as processed, got %+v", pending)
	}

	processing := store.mustGet("evt_processing_fresh")
	if processing.Status != "processing" {
		t.Fatalf("expected fresh processing event to remain processing, got %+v", processing)
	}
	if processing.ClaimedAt == nil || !processing.ClaimedAt.Equal(freshClaim) {
		t.Fatalf("expected fresh processing claim timestamp to stay untouched, got %+v", processing)
	}

	deadLetter := store.mustGet("evt_dead_letter_blocked")
	if deadLetter.Status != "dead_letter" || deadLetter.ProcessedAt != nil {
		t.Fatalf("expected dead-letter event to remain untouched, got %+v", deadLetter)
	}
}

func TestDurableBusPreservesManualRecoveryHistoryAcrossAdditionalDispatchAttempts(t *testing.T) {
	store := newOutboxTestStore()
	bus := NewDurableBus(store)

	failOnce := true
	bus.Subscribe("sync.requested", func(_ context.Context, _ types.DomainEvent) error {
		if failOnce {
			failOnce = false
			return temporaryDispatchError{err: errors.New("temporary dispatcher failure after manual retry")}
		}
		return nil
	})

	now := time.Now().UTC()
	store.events["evt_manual_retry_1"] = types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        "evt_manual_retry_1",
			CreatedAt: now.Add(-5 * time.Minute),
			UpdatedAt: now.Add(-2 * time.Minute),
			Metadata: types.Metadata{
				"manual_recovery_history": []any{
					types.Metadata{
						"action":          "retry",
						"requested_at":    now.Add(-2 * time.Minute).Format(time.RFC3339Nano),
						"previous_status": "error",
						"next_status":     "pending",
						"attempts":        2,
						"last_error":      "temporary dispatch failure",
					},
				},
				"manual_recovery_last_action":       "retry",
				"manual_recovery_last_requested_at": now.Add(-2 * time.Minute).Format(time.RFC3339Nano),
				"last_error_class":                  "temporary",
				"recovery_hint":                     "check upstream dependency health before forcing an immediate retry",
			},
		},
		EventType:      "sync.requested",
		OrganizationID: "org_123",
		ResourceType:   "integration",
		ResourceID:     "integration_123",
		Status:         "pending",
		Attempts:       2,
		LastError:      "temporary dispatch failure",
	}

	dispatched, err := bus.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched != 0 {
		t.Fatalf("expected failed retry pass to report zero successful dispatches, got %d", dispatched)
	}

	failed := store.mustGet("evt_manual_retry_1")
	if failed.Status != "error" || failed.Attempts != 3 || failed.NextAttemptAt == nil {
		t.Fatalf("expected manual retry event to return to error after handler failure, got %+v", failed)
	}
	if history, ok := failed.Metadata["manual_recovery_history"].([]any); !ok || len(history) != 1 {
		t.Fatalf("expected manual recovery history to survive additional failure, got %+v", failed.Metadata)
	}
	if failed.Metadata["last_error_class"] != "temporary" {
		t.Fatalf("expected temporary error class after failure, got %+v", failed.Metadata)
	}

	retryAt := now.Add(-1 * time.Second)
	failed.Status = "pending"
	failed.NextAttemptAt = &retryAt
	failed.UpdatedAt = retryAt
	if err := store.UpdateOutboxEvent(context.Background(), failed); err != nil {
		t.Fatal(err)
	}

	dispatched, err = bus.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched != 1 {
		t.Fatalf("expected one successful dispatch after retrying recovered event, got %d", dispatched)
	}

	processed := store.mustGet("evt_manual_retry_1")
	if processed.Status != "processed" || processed.ProcessedAt == nil || processed.Attempts != 4 {
		t.Fatalf("expected processed event after second dispatch, got %+v", processed)
	}
	if history, ok := processed.Metadata["manual_recovery_history"].([]any); !ok || len(history) != 1 {
		t.Fatalf("expected manual recovery history to remain after success, got %+v", processed.Metadata)
	}
	if _, ok := processed.Metadata["last_error_class"]; ok {
		t.Fatalf("expected transient error class metadata to clear after success, got %+v", processed.Metadata)
	}
	if _, ok := processed.Metadata["recovery_hint"]; ok {
		t.Fatalf("expected transient recovery hint metadata to clear after success, got %+v", processed.Metadata)
	}
	if processed.Metadata["manual_recovery_last_action"] != "retry" {
		t.Fatalf("expected manual recovery marker to remain intelligible after success, got %+v", processed.Metadata)
	}
}

type outboxTestStore struct {
	mu     sync.Mutex
	events map[string]types.OutboxEvent
}

type temporaryDispatchError struct {
	err error
}

func (e temporaryDispatchError) Error() string {
	return e.err.Error()
}

func (e temporaryDispatchError) Retryable() bool {
	return true
}

type permanentDispatchError struct {
	err error
}

func (e permanentDispatchError) Error() string {
	return e.err.Error()
}

func (e permanentDispatchError) Retryable() bool {
	return false
}

func newOutboxTestStore() *outboxTestStore {
	return &outboxTestStore{events: make(map[string]types.OutboxEvent)}
}

func (s *outboxTestStore) CreateOutboxEvent(_ context.Context, event types.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[event.ID] = event
	return nil
}

func (s *outboxTestStore) ClaimOutboxEvents(_ context.Context, now time.Time, limit int, staleClaimBefore time.Time) ([]types.OutboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	type candidate struct {
		id    string
		event types.OutboxEvent
	}
	candidates := make([]candidate, 0, len(s.events))
	for id, event := range s.events {
		if event.Status == "processed" || event.Status == "dead_letter" {
			continue
		}
		if event.NextAttemptAt != nil && event.NextAttemptAt.After(now) {
			continue
		}
		if event.ClaimedAt != nil && event.ClaimedAt.After(staleClaimBefore) {
			continue
		}
		candidates = append(candidates, candidate{id: id, event: event})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].event.CreatedAt.Before(candidates[j].event.CreatedAt)
	})
	items := make([]types.OutboxEvent, 0, limit)
	for _, candidate := range candidates {
		event := candidate.event
		event.Status = "processing"
		event.ClaimedAt = &now
		event.UpdatedAt = now
		s.events[candidate.id] = event
		items = append(items, event)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (s *outboxTestStore) UpdateOutboxEvent(_ context.Context, event types.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[event.ID] = event
	return nil
}

func (s *outboxTestStore) mustGet(id string) types.OutboxEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events[id]
}

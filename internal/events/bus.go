package events

import (
	"context"
	"sync"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Handler func(context.Context, types.DomainEvent) error

type Bus interface {
	Publish(context.Context, types.DomainEvent) error
	Subscribe(string, Handler)
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

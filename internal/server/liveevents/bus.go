// Package liveevents provides bounded, process-local invalidation notifications.
// Durable state stays in the repositories; clients resynchronize on connection.
package liveevents

import "sync"

type Bus struct {
	mu            sync.Mutex
	subscriptions map[string]map[chan struct{}]struct{}
}

func (b *Bus) Subscribe(topic string) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	if b == nil {
		return ch, func() {}
	}
	b.mu.Lock()
	if b.subscriptions == nil {
		b.subscriptions = make(map[string]map[chan struct{}]struct{})
	}
	if b.subscriptions[topic] == nil {
		b.subscriptions[topic] = make(map[chan struct{}]struct{})
	}
	b.subscriptions[topic][ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subscriptions[topic], ch)
		if len(b.subscriptions[topic]) == 0 {
			delete(b.subscriptions, topic)
		}
	}
}

func (b *Bus) Publish(topic string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscriptions[topic] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

package liveevents

import "testing"

func TestSubscriptionsAreScopedBoundedAndRemovable(t *testing.T) {
	bus := &Bus{}
	a, stop := bus.Subscribe("user:a")
	b, stopB := bus.Subscribe("user:b")
	defer stopB()
	for range 100 {
		bus.Publish("user:a")
	}
	select {
	case <-a:
	default:
		t.Fatal("missing invalidation")
	}
	select {
	case <-a:
		t.Fatal("notifications must coalesce")
	default:
	}
	select {
	case <-b:
		t.Fatal("cross-user notification")
	default:
	}
	stop()
	stop()
	bus.Publish("user:a")
	select {
	case <-a:
		t.Fatal("removed subscriber notified")
	default:
	}
}

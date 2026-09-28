package liveevents

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

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

func TestChangesRetainDifferentDocumentsAndCoalesceRecipients(t *testing.T) {
	bus := &Bus{}
	bus.PublishChange(Change{DocumentID: "offline", Resource: "files"})
	wake, drain, stop := bus.SubscribeChanges()
	require.Empty(t, drain())
	recipient := uuid.New()
	for range 1000 {
		bus.PublishChange(Change{DocumentID: "a", Resource: "assignments", PreviousReaders: []uuid.UUID{recipient}, VisibilityChanged: true})
	}
	bus.PublishChange(Change{DocumentID: "a", Resource: "assignments"})
	bus.PublishChange(Change{DocumentID: "b", Resource: "files"})
	<-wake
	changes := drain()
	require.Len(t, changes, 2)
	for _, change := range changes {
		if change.DocumentID == "a" {
			require.True(t, change.VisibilityChanged)
			require.Equal(t, []uuid.UUID{recipient}, change.PreviousReaders)
		}
	}
	stop()
	stop()
	bus.PublishChange(Change{DocumentID: "c", Resource: "files"})
	require.Empty(t, drain())
}

func TestSlowChangeConsumerResynchronizesAfterOverflow(t *testing.T) {
	bus := &Bus{}
	wake, drain, stop := bus.SubscribeChanges()
	defer stop()
	for range maxPendingChanges + 1 {
		bus.PublishChange(Change{DocumentID: uuid.NewString(), Resource: "files"})
	}
	<-wake
	require.Equal(t, []Change{{Resource: "resync"}}, drain())
	bus.PublishChange(Change{DocumentID: "next", Resource: "files"})
	<-wake
	require.Equal(t, []Change{{DocumentID: "next", Resource: "files"}}, drain())
}

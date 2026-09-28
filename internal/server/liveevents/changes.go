package liveevents

import "github.com/google/uuid"

// Change invalidates a document resource; it never carries document contents.
type Change struct {
	DocumentKind      string `json:"documentKind,omitempty"`
	DocumentID        string `json:"documentId,omitempty"`
	Resource          string `json:"resource"`
	VisibilityChanged bool   `json:"visibilityChanged,omitempty"`
	// PreviousReaders also receive the final invalidation after losing implicit access.
	PreviousReaders []uuid.UUID `json:"-"`
	// A nil audience means a capability change can affect every active session.
	Audience []uuid.UUID `json:"-"`
}

const maxPendingChanges = 256

type changeSubscription struct {
	wake    chan struct{}
	pending map[string]Change
}

// SubscribeChanges registers only a live consumer. Pending changes coalesce by
// document and resource. Overflow becomes a resync rather than losing updates.
func (b *Bus) SubscribeChanges() (<-chan struct{}, func() []Change, func()) {
	s := &changeSubscription{wake: make(chan struct{}, 1), pending: make(map[string]Change)}
	b.mu.Lock()
	if b.changes == nil {
		b.changes = make(map[*changeSubscription]struct{})
	}
	b.changes[s] = struct{}{}
	b.mu.Unlock()
	drain := func() []Change {
		b.mu.Lock()
		defer b.mu.Unlock()
		result := make([]Change, 0, len(s.pending))
		for _, change := range s.pending {
			result = append(result, change)
		}
		clear(s.pending)
		return result
	}
	stop := func() { b.mu.Lock(); defer b.mu.Unlock(); delete(b.changes, s) }
	return s.wake, drain, stop
}

func (b *Bus) PublishChange(change Change) {
	if len(change.PreviousReaders) > maxPendingChanges || len(change.Audience) > maxPendingChanges {
		change = Change{Resource: "resync"}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for s := range b.changes {
		if _, overflow := s.pending["resync"]; !overflow {
			key := change.DocumentID + ":" + change.Resource
			if previous, ok := s.pending[key]; ok {
				changeForSubscriber := change
				changeForSubscriber.VisibilityChanged = previous.VisibilityChanged || change.VisibilityChanged
				changeForSubscriber.PreviousReaders = mergeUserIDs(previous.PreviousReaders, change.PreviousReaders)
				if len(previous.Audience) == 0 || len(change.Audience) == 0 {
					changeForSubscriber.Audience = nil
				} else {
					changeForSubscriber.Audience = mergeUserIDs(previous.Audience, change.Audience)
				}
				if len(changeForSubscriber.PreviousReaders) > maxPendingChanges || len(changeForSubscriber.Audience) > maxPendingChanges {
					clear(s.pending)
					s.pending["resync"] = Change{Resource: "resync"}
				} else {
					s.pending[key] = changeForSubscriber
				}
			} else if len(s.pending) >= maxPendingChanges {
				clear(s.pending)
				s.pending["resync"] = Change{Resource: "resync"}
			} else {
				s.pending[key] = change
			}
		}
		select {
		case s.wake <- struct{}{}:
		default:
		}
	}
}

func mergeUserIDs(a, b []uuid.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(a)+len(b))
	seen := make(map[uuid.UUID]struct{}, len(a)+len(b))
	for _, ids := range [][]uuid.UUID{a, b} {
		for _, id := range ids {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				result = append(result, id)
			}
		}
	}
	return result
}

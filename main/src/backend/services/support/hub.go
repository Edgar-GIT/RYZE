package support

import (
	"errors"
	"sync"
)

// StaffScope is the hub scope of support staff streams (every conversation).
const StaffScope = "*staff*"

const (
	// maxStreamsPerScope bounds the live streams one account (or the staff
	// scope) may hold on this instance, e.g. a few open tabs.
	maxStreamsPerScope = 3
	maxStaffStreams    = 10
	// maxStreams bounds the live streams of this instance.
	maxStreams = 500
)

// ErrTooManyStreams is returned when a scope or the instance is at its limit.
var ErrTooManyStreams = errors.New("too many live streams")

// Hub wakes the live streams of this API instance when a message is stored.
// It carries no data: each stream re-reads the messages it may see from the
// database, so a wake-up can never leak a message to the wrong reader. A
// message wakes only its account's streams and the staff streams. Streams
// also re-check the database on a timer, which covers messages stored by
// another API instance.
type Hub struct {
	mu          sync.Mutex
	subscribers map[chan struct{}]string
	perScope    map[string]int
}

// NewHub returns an empty hub.
func NewHub() *Hub {
	return &Hub{subscribers: map[chan struct{}]string{}, perScope: map[string]int{}}
}

// Subscribe registers a stream for a scope (an account id, or StaffScope) and
// returns its wake-up channel and the function that releases it.
func (h *Hub) Subscribe(scope string) (<-chan struct{}, func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	limit := maxStreamsPerScope
	if scope == StaffScope {
		limit = maxStaffStreams
	}
	if len(h.subscribers) >= maxStreams || h.perScope[scope] >= limit {
		return nil, nil, ErrTooManyStreams
	}
	ch := make(chan struct{}, 1)
	h.subscribers[ch] = scope
	h.perScope[scope]++
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			delete(h.subscribers, ch)
			h.perScope[scope]--
			if h.perScope[scope] <= 0 {
				delete(h.perScope, scope)
			}
		})
	}, nil
}

// Publish wakes the streams of the account that owns the conversation and the
// staff streams, without blocking; a stream that has not consumed its
// previous wake-up already has one pending.
func (h *Hub) Publish(ownerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch, scope := range h.subscribers {
		if scope != ownerID && scope != StaffScope {
			continue
		}
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Subscribers reports the number of live streams (for tests).
func (h *Hub) Subscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

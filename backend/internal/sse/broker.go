// Package sse is an in-process publish/subscribe broker for order-status
// events. It is deliberately single-process (no Redis, no message queue): the
// real-time feature is single-instance, as the TRD states. The HTTP layer turns
// a subscription into a Server-Sent-Events stream.
package sse

import "sync"

// StatusEvent is one order-status update pushed to subscribers. Its JSON shape
// is the SSE `data` payload the frontend consumes.
type StatusEvent struct {
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
}

// subBuffer is the per-subscriber channel capacity. A buffer lets Publish stay
// non-blocking across a brief consumer stall; if a subscriber falls further
// behind than this, the surplus event is dropped rather than blocking the
// transition that produced it (the next event, or the SSE reconnect snapshot,
// re-synchronizes the client).
const subBuffer = 16

// Broker fans status events out to the subscribers of a given order. It is safe
// for concurrent use.
type Broker struct {
	mu   sync.Mutex
	subs map[int64]map[chan StatusEvent]struct{}
}

// NewBroker returns an empty broker.
func NewBroker() *Broker {
	return &Broker{subs: make(map[int64]map[chan StatusEvent]struct{})}
}

// Subscribe registers interest in one order's events and returns a receive
// channel plus an unsubscribe func. The caller MUST call unsubscribe (e.g. via
// defer) when the stream ends; it removes the subscriber and closes the
// channel, so there is no goroutine or map leak. unsubscribe is idempotent.
func (b *Broker) Subscribe(orderID int64) (<-chan StatusEvent, func()) {
	ch := make(chan StatusEvent, subBuffer)

	b.mu.Lock()
	set := b.subs[orderID]
	if set == nil {
		set = make(map[chan StatusEvent]struct{})
		b.subs[orderID] = set
	}
	set[ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			if set := b.subs[orderID]; set != nil {
				if _, ok := set[ch]; ok {
					delete(set, ch)
					close(ch)
				}
				if len(set) == 0 {
					delete(b.subs, orderID)
				}
			}
			b.mu.Unlock()
		})
	}
	return ch, unsubscribe
}

// Publish delivers an event to every current subscriber of ev.OrderID. The send
// is non-blocking: a subscriber whose buffer is full misses this event rather
// than stalling the publisher (and thus the order transition).
func (b *Broker) Publish(ev StatusEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[ev.OrderID] {
		select {
		case ch <- ev:
		default:
		}
	}
}

// subscriberCount reports how many subscribers an order currently has. It backs
// the leak assertions in the tests.
func (b *Broker) subscriberCount(orderID int64) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs[orderID])
}

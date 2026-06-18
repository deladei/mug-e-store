package sse

import (
	"sync"
	"testing"
	"time"
)

func TestBroker_PublishReachesSubscriber(t *testing.T) {
	b := NewBroker()
	ch, unsub := b.Subscribe(1)
	defer unsub()

	b.Publish(StatusEvent{OrderID: 1, Status: "paid"})

	select {
	case ev := <-ch:
		if ev.OrderID != 1 || ev.Status != "paid" {
			t.Errorf("got %+v, want {1 paid}", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive the event")
	}
}

func TestBroker_OnlyMatchingOrder(t *testing.T) {
	b := NewBroker()
	ch1, unsub1 := b.Subscribe(1)
	defer unsub1()
	ch2, unsub2 := b.Subscribe(2)
	defer unsub2()

	b.Publish(StatusEvent{OrderID: 1, Status: "ready"})

	select {
	case <-ch1:
	case <-time.After(time.Second):
		t.Fatal("subscriber for order 1 missed its event")
	}
	select {
	case ev := <-ch2:
		t.Fatalf("subscriber for order 2 wrongly received %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// expected: no event for order 2
	}
}

func TestBroker_UnsubscribeStopsDelivery(t *testing.T) {
	b := NewBroker()
	ch, unsub := b.Subscribe(1)
	unsub()

	// The channel must be closed and the order's subscriber set cleaned up.
	if _, ok := <-ch; ok {
		t.Error("channel was not closed on unsubscribe")
	}
	if b.subscriberCount(1) != 0 {
		t.Errorf("subscriberCount(1) = %d after unsubscribe, want 0", b.subscriberCount(1))
	}

	// Publishing after everyone left must not panic.
	b.Publish(StatusEvent{OrderID: 1, Status: "completed"})
}

func TestBroker_UnsubscribeIsIdempotent(t *testing.T) {
	b := NewBroker()
	_, unsub := b.Subscribe(1)
	unsub()
	unsub() // second call must be a safe no-op, not a double-close panic
}

func TestBroker_SlowSubscriberDoesNotBlock(t *testing.T) {
	b := NewBroker()
	// Never drain this subscriber; publishing must not block past the buffer.
	_, unsub := b.Subscribe(1)
	defer unsub()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			b.Publish(StatusEvent{OrderID: 1, Status: "preparing"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}
}

// TestBroker_ConcurrentSubscribePublish is meant to be run under -race: many
// goroutines subscribing, publishing, and unsubscribing at once must not race
// or leak.
func TestBroker_ConcurrentSubscribePublish(t *testing.T) {
	b := NewBroker()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			ch, unsub := b.Subscribe(id % 5)
			defer unsub()
			go b.Publish(StatusEvent{OrderID: id % 5, Status: "paid"})
			select {
			case <-ch:
			case <-time.After(200 * time.Millisecond):
			}
		}(int64(i))
	}
	wg.Wait()
	for order := int64(0); order < 5; order++ {
		if n := b.subscriberCount(order); n != 0 {
			t.Errorf("order %d leaked %d subscribers", order, n)
		}
	}
}

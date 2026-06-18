package store

import (
	"context"
	"errors"
	"testing"

	"coffeemug/backend/internal/domain"
)

// payOrder drives an order from pending_payment to paid the way only the webhook
// legitimately can, so transition tests can reach mid-lifecycle states.
func payOrder(t *testing.T, st *Store, orderID int64) {
	t.Helper()
	if _, err := st.TransitionOrder(context.Background(), orderID, domain.StatusPaid, nil); err != nil {
		t.Fatalf("transition to paid: %v", err)
	}
}

func TestTransition_LegalChain(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "legal@example.com")
	v := seedVariant(t, st, 1000)
	order := pickupOrder(t, st, u.ID, v, 1, 0)

	actor := u.ID
	for _, to := range []domain.Status{domain.StatusPaid, domain.StatusPreparing, domain.StatusReady, domain.StatusCompleted} {
		got, err := st.TransitionOrder(ctx, order.ID, to, &actor)
		if err != nil {
			t.Fatalf("transition to %s: %v", to, err)
		}
		if got.Status != to {
			t.Fatalf("status = %s, want %s", got.Status, to)
		}
	}
}

func TestTransition_IllegalIsRejected(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "illegal@example.com")
	v := seedVariant(t, st, 1000)
	order := pickupOrder(t, st, u.ID, v, 1, 0)

	// pending_payment → ready is not an edge in the lifecycle graph.
	actor := u.ID
	_, err := st.TransitionOrder(ctx, order.ID, domain.StatusReady, &actor)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("err = %v, want ErrInvalidTransition", err)
	}
}

func TestTransition_NoOpIsIdempotent(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "noop@example.com")
	v := seedVariant(t, st, 1000)
	order := pickupOrder(t, st, u.ID, v, 1, 0)
	payOrder(t, st, order.ID)

	// Re-applying paid (as a retried webhook would) is a successful no-op, and it
	// must not write a second order_events row.
	got, err := st.TransitionOrder(ctx, order.ID, domain.StatusPaid, nil)
	if err != nil {
		t.Fatalf("no-op transition: %v", err)
	}
	if got.Status != domain.StatusPaid {
		t.Errorf("status = %s, want paid", got.Status)
	}
	events, err := st.GetOrderEvents(ctx, order.ID)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("order_events rows = %d, want 1 (no-op must not add an event)", len(events))
	}
}

func TestTransition_LoyaltyEarnedOnCompletion(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "earn@example.com")
	v := seedVariant(t, st, 2550) // GHS 25.50 subtotal → 25 whole-GHS points
	order := pickupOrder(t, st, u.ID, v, 1, 0)

	actor := u.ID
	for _, to := range []domain.Status{domain.StatusPaid, domain.StatusPreparing, domain.StatusReady, domain.StatusCompleted} {
		if _, err := st.TransitionOrder(ctx, order.ID, to, &actor); err != nil {
			t.Fatalf("transition to %s: %v", to, err)
		}
	}
	bal, err := st.LoyaltyBalance(ctx, u.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 25 {
		t.Errorf("earned points = %d, want 25 (subtotal 2550 / 100)", bal)
	}
}

// TestTransition_RefundOnCancel covers PRD §S1: cancelling an order that redeemed
// points returns them via a compensating ledger entry. This also protects the
// checkout cancelDangling path.
func TestTransition_RefundOnCancel(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "refund@example.com")
	v := seedVariant(t, st, 1000)
	grantPoints(t, st, u.ID, 400)

	order := pickupOrder(t, st, u.ID, v, 1, 400)
	// Spent down to zero by the redemption.
	if bal, _ := st.LoyaltyBalance(ctx, u.ID); bal != 0 {
		t.Fatalf("balance after redeem = %d, want 0", bal)
	}

	if _, err := st.TransitionOrder(ctx, order.ID, domain.StatusCancelled, nil); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	// The 400 redeemed points are refunded.
	bal, err := st.LoyaltyBalance(ctx, u.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 400 {
		t.Errorf("balance after cancel = %d, want 400 (points refunded)", bal)
	}
}

func TestTransition_NoRefundWhenNothingRedeemed(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "norefund@example.com")
	v := seedVariant(t, st, 1000)
	order := pickupOrder(t, st, u.ID, v, 1, 0) // no redemption

	if _, err := st.TransitionOrder(ctx, order.ID, domain.StatusCancelled, nil); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	bal, err := st.LoyaltyBalance(ctx, u.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 0 {
		t.Errorf("balance = %d, want 0 (no redemption, so no refund entry)", bal)
	}
}
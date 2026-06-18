package store

import (
	"context"
	"errors"
	"sync"
	"testing"

	"coffeemug/backend/internal/domain"
)

func TestCheckout_HappyPath(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "happy@example.com")
	v := seedVariant(t, st, 1500) // GHS 15.00

	order := pickupOrder(t, st, u.ID, v, 2, 0)

	if order.Status != domain.StatusPendingPayment {
		t.Errorf("status = %s, want pending_payment", order.Status)
	}
	if order.SubtotalPesewas != 3000 {
		t.Errorf("subtotal = %d, want 3000", order.SubtotalPesewas)
	}
	if order.DeliveryFeePesewas != 0 {
		t.Errorf("pickup delivery fee = %d, want 0", order.DeliveryFeePesewas)
	}
	if order.TotalPesewas != 3000 {
		t.Errorf("total = %d, want 3000", order.TotalPesewas)
	}
	if len(order.Lines) != 1 || order.Lines[0].Quantity != 2 {
		t.Fatalf("expected one line of qty 2, got %+v", order.Lines)
	}
	// The cart must be emptied by a successful checkout.
	cart, err := st.GetCart(ctx, u.ID)
	if err != nil {
		t.Fatalf("get cart: %v", err)
	}
	if len(cart.Lines) != 0 {
		t.Errorf("cart not cleared: %+v", cart.Lines)
	}
}

func TestCheckout_DeliveryAddsFee(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "delivery@example.com")
	v := seedVariant(t, st, 1000)

	if _, err := st.AddCartItem(ctx, u.ID, v, 1); err != nil {
		t.Fatalf("add to cart: %v", err)
	}
	order, err := st.Checkout(ctx, CheckoutParams{
		UserID:             u.ID,
		Fulfilment:         domain.FulfilmentDelivery,
		Address:            "12 Oxford St, Osu",
		Phone:              "0240000000",
		DeliveryFeePesewas: 1000,
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if order.DeliveryFeePesewas != 1000 || order.TotalPesewas != 2000 {
		t.Errorf("delivery fee/total = %d/%d, want 1000/2000", order.DeliveryFeePesewas, order.TotalPesewas)
	}
}

func TestCheckout_EmptyCart(t *testing.T) {
	st := testStore(t)
	u := seedUser(t, st, "empty@example.com")

	_, err := st.Checkout(context.Background(), CheckoutParams{UserID: u.ID, Fulfilment: domain.FulfilmentPickup})
	if !errors.Is(err, ErrEmptyCart) {
		t.Fatalf("err = %v, want ErrEmptyCart", err)
	}
}

func TestCheckout_UnavailableItem(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "unavail@example.com")
	v := seedVariant(t, st, 1000)
	if _, err := st.AddCartItem(ctx, u.ID, v, 1); err != nil {
		t.Fatalf("add to cart: %v", err)
	}
	// Turn the item off after it is in the cart.
	item, err := st.GetItem(ctx, 1, false)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if _, err := st.SetItemAvailability(ctx, item.ID, false); err != nil {
		t.Fatalf("set unavailable: %v", err)
	}

	_, err = st.Checkout(ctx, CheckoutParams{UserID: u.ID, Fulfilment: domain.FulfilmentPickup})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestCheckout_DuplicateIdempotencyKey(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "idem@example.com")
	v := seedVariant(t, st, 1000)

	if _, err := st.AddCartItem(ctx, u.ID, v, 1); err != nil {
		t.Fatalf("add to cart: %v", err)
	}
	if _, err := st.Checkout(ctx, CheckoutParams{UserID: u.ID, Fulfilment: domain.FulfilmentPickup, IdempotencyKey: "abc-123"}); err != nil {
		t.Fatalf("first checkout: %v", err)
	}
	// Re-add and retry the same idempotency key — must be rejected.
	if _, err := st.AddCartItem(ctx, u.ID, v, 1); err != nil {
		t.Fatalf("re-add to cart: %v", err)
	}
	_, err := st.Checkout(ctx, CheckoutParams{UserID: u.ID, Fulfilment: domain.FulfilmentPickup, IdempotencyKey: "abc-123"})
	if !errors.Is(err, ErrDuplicateOrder) {
		t.Fatalf("err = %v, want ErrDuplicateOrder", err)
	}
}

func TestCheckout_RedeemExactBalance(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "redeem@example.com")
	v := seedVariant(t, st, 1000) // subtotal 1000
	grantPoints(t, st, u.ID, 400) // worth 400 pesewas

	order := pickupOrder(t, st, u.ID, v, 1, 400)

	if order.DiscountPesewas != 400 {
		t.Errorf("discount = %d, want 400", order.DiscountPesewas)
	}
	if order.TotalPesewas != 600 {
		t.Errorf("total = %d, want 600 (1000 - 400)", order.TotalPesewas)
	}
	// Balance must be fully spent.
	bal, err := st.LoyaltyBalance(ctx, u.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 0 {
		t.Errorf("balance after redeeming all = %d, want 0", bal)
	}
}

func TestCheckout_RedeemMoreThanBalanceRejected(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "greedy@example.com")
	v := seedVariant(t, st, 1000)
	grantPoints(t, st, u.ID, 100)

	if _, err := st.AddCartItem(ctx, u.ID, v, 1); err != nil {
		t.Fatalf("add to cart: %v", err)
	}
	_, err := st.Checkout(ctx, CheckoutParams{UserID: u.ID, Fulfilment: domain.FulfilmentPickup, RedeemPoints: 101})
	if !errors.Is(err, ErrInsufficientPoints) {
		t.Fatalf("err = %v, want ErrInsufficientPoints", err)
	}
	// The rejected checkout must not have spent any points.
	bal, err := st.LoyaltyBalance(ctx, u.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 100 {
		t.Errorf("balance after rejected redemption = %d, want 100 (untouched)", bal)
	}
}

// TestCheckout_ConcurrentRedeemDoesNotDoubleSpend is the test the FOR UPDATE
// user-row lock exists to pass: two checkouts racing to redeem the same points
// must not both succeed. The user has exactly enough points for one of the two
// orders to redeem; the other must be rejected with ErrInsufficientPoints.
func TestCheckout_ConcurrentRedeemDoesNotDoubleSpend(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "race@example.com")
	v := seedVariant(t, st, 1000)
	grantPoints(t, st, u.ID, 500) // enough for exactly ONE 500-point redemption

	// Two separate carts' worth: add qty 2 so each checkout has a line to buy.
	if _, err := st.AddCartItem(ctx, u.ID, v, 1); err != nil {
		t.Fatalf("add to cart: %v", err)
	}

	// Both goroutines try to redeem all 500 points at once. Because each checkout
	// reads the cart and the balance inside its own transaction, and the cart is
	// cleared on success, exactly one can win the redemption; we assert the
	// points were spent at most once regardless of which succeeds.
	var wg sync.WaitGroup
	results := make([]error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			_, err := st.Checkout(ctx, CheckoutParams{
				UserID:       u.ID,
				Fulfilment:   domain.FulfilmentPickup,
				RedeemPoints: 500,
			})
			results[idx] = err
		}(i)
	}
	close(start)
	wg.Wait()

	// Exactly one redemption may commit: the winner spends all 500 points, the
	// loser is serialized behind the user-row lock and then sees a zero balance.
	// So the ending balance must be exactly 0 — a double-spend would leave it at
	// -500 (both debited), and "nobody redeemed" (impossible here) would leave 500.
	bal, err := st.LoyaltyBalance(ctx, u.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 0 {
		t.Fatalf("balance = %d, want 0 (a non-zero value means the lock failed to serialize the redemption)", bal)
	}
	// The loser must fail cleanly (insufficient points, or an empty cart if it
	// was fully serialized after the winner cleared it) — never a corrupt state.
	successes := 0
	for i, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInsufficientPoints), errors.Is(err, ErrEmptyCart):
			// expected loser outcomes
		default:
			t.Errorf("goroutine %d: unexpected error %v", i, err)
		}
	}
	if successes != 1 {
		t.Errorf("successful checkouts = %d, want exactly 1", successes)
	}
}

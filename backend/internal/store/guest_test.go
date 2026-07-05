package store

import (
	"context"
	"testing"

	"coffeemug/backend/internal/domain"
)

// seedGuest inserts a passwordless guest account (PRD S4) and returns it.
func seedGuest(t *testing.T, st *Store, email string) *User {
	t.Helper()
	u := &User{Name: "Guest", Email: email, Phone: "0240000000", PasswordHash: "x", Role: "customer", IsGuest: true}
	if err := st.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("seed guest: %v", err)
	}
	return u
}

// TestCreateUser_IsGuestRoundTrips proves the flag is written and read back: a
// guest is stored is_guest=true and a normal customer false, so downstream code
// (loyalty earn) can tell them apart.
func TestCreateUser_IsGuestRoundTrips(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	guest := seedGuest(t, st, "guest-roundtrip@example.com")
	got, err := st.GetUserByID(ctx, guest.ID)
	if err != nil {
		t.Fatalf("get guest: %v", err)
	}
	if !got.IsGuest {
		t.Error("guest GetUserByID IsGuest = false, want true")
	}

	customer := seedUser(t, st, "customer-roundtrip@example.com")
	got, err = st.GetUserByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("get customer: %v", err)
	}
	if got.IsGuest {
		t.Error("customer GetUserByID IsGuest = true, want false")
	}
}

// TestTransition_GuestEarnsNoLoyalty covers the S4 rule (DECISIONS 2026-06-16): a
// completed guest order earns nothing, because a guest can never log back in to
// spend points. The order otherwise drives the full lifecycle exactly like a
// registered customer's — only the earn ledger row is suppressed.
func TestTransition_GuestEarnsNoLoyalty(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedGuest(t, st, "guest-earn@example.com")
	v := seedVariant(t, st, 2550) // would earn 25 points for a registered user
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
	if bal != 0 {
		t.Errorf("guest earned points = %d, want 0 (guests earn no loyalty)", bal)
	}
}

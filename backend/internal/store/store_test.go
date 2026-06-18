package store

import (
	"context"
	"os"
	"testing"

	"coffeemug/backend/internal/domain"
)

// These tests need a real Postgres (the store is the only SQL-speaking layer, so
// there is nothing meaningful to test against a mock). Point TEST_DATABASE_URL at
// a throwaway database with the 0001_init schema applied; the suite truncates
// every table before each test, so it must never point at real data.
//
//	createdb coffeemug_test
//	psql -d coffeemug_test -f migrations/0001_init.up.sql
//	TEST_DATABASE_URL='host=/var/run/postgresql user=USER dbname=coffeemug_test sslmode=disable' \
//	  go test -p 1 ./...
//
// With the variable unset the suite skips, so `go test ./...` stays green on a
// machine without Postgres. The store and httpapi suites share the one test
// database and truncate it, so run the whole tree with -p 1 (serialize the
// package binaries); within a package tests are already serial.

// testStore opens the test database and hands back a clean Store: every table is
// truncated first, so each test starts from an empty, identity-reset schema.
func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run store tests against Postgres")
	}
	st, err := Open(dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	truncateAll(t, st)
	return st
}

// truncateAll wipes every table and resets identity sequences. CASCADE handles
// the foreign-key order so the list need not be topologically sorted.
func truncateAll(t *testing.T, st *Store) {
	t.Helper()
	const q = `TRUNCATE
		loyalty_ledger, order_events, order_lines, orders,
		cart_lines, carts, item_variants, items, categories,
		password_reset_tokens, refresh_tokens, users
		RESTART IDENTITY CASCADE`
	if _, err := st.db.ExecContext(context.Background(), q); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedUser inserts a customer and returns it. Email must be unique per test.
func seedUser(t *testing.T, st *Store, email string) *User {
	t.Helper()
	u := &User{Name: "Kwame Mensah", Email: email, Phone: "0240000000", PasswordHash: "x", Role: "customer"}
	if err := st.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

// seedVariant creates a category, an item and one priced variant, returning the
// variant id — the smallest catalog needed to put something in a cart.
func seedVariant(t *testing.T, st *Store, pricePesewas int64) int64 {
	t.Helper()
	ctx := context.Background()
	cat := &Category{Name: "Coffee", SortOrder: 1}
	if err := st.CreateCategory(ctx, cat); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	item := &Item{CategoryID: cat.ID, Name: "Flat White", Description: "", ImageURL: ""}
	if err := st.CreateItem(ctx, item); err != nil {
		t.Fatalf("seed item: %v", err)
	}
	v := &Variant{ItemID: item.ID, Name: "Regular", PricePesewas: pricePesewas, SortOrder: 1}
	if err := st.CreateVariant(ctx, v); err != nil {
		t.Fatalf("seed variant: %v", err)
	}
	return v.ID
}

// grantPoints credits a user's ledger directly, so a test can set up a balance to
// redeem without driving a full earn-on-completion flow.
func grantPoints(t *testing.T, st *Store, userID, points int64) {
	t.Helper()
	const q = `INSERT INTO loyalty_ledger (user_id, delta, reason) VALUES ($1, $2, 'test_grant')`
	if _, err := st.db.ExecContext(context.Background(), q, userID, points); err != nil {
		t.Fatalf("grant points: %v", err)
	}
}

// pickupOrder builds a cart with one line at the given price×qty and checks out a
// pickup order, optionally redeeming points. It returns the created order.
func pickupOrder(t *testing.T, st *Store, userID, variantID int64, qty int, redeem int64) *Order {
	t.Helper()
	ctx := context.Background()
	if _, err := st.AddCartItem(ctx, userID, variantID, qty); err != nil {
		t.Fatalf("add to cart: %v", err)
	}
	order, err := st.Checkout(ctx, CheckoutParams{
		UserID:       userID,
		Fulfilment:   domain.FulfilmentPickup,
		RedeemPoints: redeem,
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	return order
}

package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

// countRows is a tiny helper for the assertions below.
func countRows(t *testing.T, st *Store, q string, args ...any) int {
	t.Helper()
	var n int
	if err := st.db.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestConsumePasswordReset_HappyPath(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "ama@example.com")

	// A live session and a live reset token before the reset.
	if err := st.CreateRefreshToken(ctx, u.ID, "refreshhash", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create refresh: %v", err)
	}
	if err := st.CreatePasswordResetToken(ctx, u.ID, "resethash", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create reset: %v", err)
	}

	if err := st.ConsumePasswordReset(ctx, "resethash", "new-bcrypt-hash"); err != nil {
		t.Fatalf("consume: %v", err)
	}

	// Password changed.
	got, err := st.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.PasswordHash != "new-bcrypt-hash" {
		t.Fatalf("password hash not updated: %q", got.PasswordHash)
	}
	// Reset tokens cleared and sessions revoked.
	if n := countRows(t, st, `SELECT count(*) FROM password_reset_tokens WHERE user_id = $1`, u.ID); n != 0 {
		t.Fatalf("reset tokens not cleared: %d", n)
	}
	if n := countRows(t, st, `SELECT count(*) FROM refresh_tokens WHERE user_id = $1`, u.ID); n != 0 {
		t.Fatalf("sessions not revoked: %d", n)
	}
}

func TestConsumePasswordReset_SingleUse(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "kojo@example.com")
	if err := st.CreatePasswordResetToken(ctx, u.ID, "onceonly", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create reset: %v", err)
	}
	if err := st.ConsumePasswordReset(ctx, "onceonly", "hash1"); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := st.ConsumePasswordReset(ctx, "onceonly", "hash2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("replay should be ErrNotFound, got %v", err)
	}
}

func TestConsumePasswordReset_Unknown(t *testing.T) {
	st := testStore(t)
	if err := st.ConsumePasswordReset(context.Background(), "never-existed", "hash"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestConsumePasswordReset_Expired(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "esi@example.com")
	if err := st.CreatePasswordResetToken(ctx, u.ID, "staletoken", time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("create reset: %v", err)
	}
	if err := st.ConsumePasswordReset(ctx, "staletoken", "hash"); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("want ErrTokenExpired, got %v", err)
	}
	// An expired-but-rejected token leaves the password untouched.
	got, _ := st.GetUserByID(ctx, u.ID)
	if got.PasswordHash != "x" {
		t.Fatalf("password should be unchanged, got %q", got.PasswordHash)
	}
}

package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"coffeemug/backend/internal/auth"
)

// refreshTokenCount reports how many live sessions a user has.
func (h *harness) refreshTokenCount(t *testing.T, userID int64) int {
	t.Helper()
	var n int
	if err := h.store.DB().QueryRowContext(context.Background(),
		`SELECT count(*) FROM refresh_tokens WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	return n
}

// registerUser registers a customer and returns the decoded auth response.
func registerUser(t *testing.T, h *harness, email, password string) authResponse {
	t.Helper()
	resp := h.request(t, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Akosua Asante", "email": email, "phone": "0240000000", "password": password,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", resp.StatusCode)
	}
	var out authResponse
	decodeJSONBody(t, resp, &out)
	return out
}

// TestPasswordResetRequest_GenericRegardlessOfEmail proves the request endpoint
// never reveals whether an account exists: a registered email, an unknown email,
// and a malformed email all return the same 200.
func TestPasswordResetRequest_GenericRegardlessOfEmail(t *testing.T) {
	h := newHarness(t)
	registerUser(t, h, "real@example.com", "supersecret")

	for _, email := range []string{"real@example.com", "ghost@example.com", "not-an-email"} {
		resp := h.request(t, http.MethodPost, "/api/v1/auth/password-reset/request", "",
			map[string]string{"email": email})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("email %q: status = %d, want 200", email, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// TestPasswordResetConfirm_HappyPath redeems a valid token: the password is
// changed (old fails, new works at login) and every existing session is revoked.
func TestPasswordResetConfirm_HappyPath(t *testing.T) {
	h := newHarness(t)
	reg := registerUser(t, h, "switch@example.com", "oldpassword")

	// Register opened a session; prove it gets revoked by the reset.
	if n := h.refreshTokenCount(t, reg.User.ID); n == 0 {
		t.Fatal("expected a session after register")
	}

	// Stand in for the email: insert a token with a raw value we control.
	const raw = "known-raw-reset-token-value"
	if err := h.store.CreatePasswordResetToken(context.Background(), reg.User.ID,
		auth.HashToken(raw), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("seed reset token: %v", err)
	}

	resp := h.request(t, http.MethodPost, "/api/v1/auth/password-reset/confirm", "",
		map[string]string{"token": raw, "password": "newpassword"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// The reset revoked every existing session (check before any new login).
	if n := h.refreshTokenCount(t, reg.User.ID); n != 0 {
		t.Fatalf("sessions after reset = %d, want 0", n)
	}

	// Old password rejected, new password accepted.
	resp = h.request(t, http.MethodPost, "/api/v1/auth/login", "",
		map[string]string{"email": "switch@example.com", "password": "oldpassword"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old password login = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	resp = h.request(t, http.MethodPost, "/api/v1/auth/login", "",
		map[string]string{"email": "switch@example.com", "password": "newpassword"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("new password login = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPasswordResetConfirm_BadTokenRejected(t *testing.T) {
	h := newHarness(t)
	resp := h.request(t, http.MethodPost, "/api/v1/auth/password-reset/confirm", "",
		map[string]string{"token": "this-token-was-never-issued", "password": "newpassword"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if code := decodeError(t, resp); code != codeInvalidToken {
		t.Errorf("code = %q, want %q", code, codeInvalidToken)
	}
}

func TestPasswordResetConfirm_ExpiredTokenRejected(t *testing.T) {
	h := newHarness(t)
	reg := registerUser(t, h, "stale@example.com", "oldpassword")
	const raw = "expired-raw-token"
	if err := h.store.CreatePasswordResetToken(context.Background(), reg.User.ID,
		auth.HashToken(raw), time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("seed reset token: %v", err)
	}
	resp := h.request(t, http.MethodPost, "/api/v1/auth/password-reset/confirm", "",
		map[string]string{"token": raw, "password": "newpassword"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if code := decodeError(t, resp); code != codeInvalidToken {
		t.Errorf("code = %q, want %q", code, codeInvalidToken)
	}
}

func TestPasswordResetConfirm_ShortPasswordRejected(t *testing.T) {
	h := newHarness(t)
	resp := h.request(t, http.MethodPost, "/api/v1/auth/password-reset/confirm", "",
		map[string]string{"token": "whatever", "password": "short"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if code := decodeError(t, resp); code != codeValidation {
		t.Errorf("code = %q, want %q", code, codeValidation)
	}
}

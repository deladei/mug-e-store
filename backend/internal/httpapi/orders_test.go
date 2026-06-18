package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func orderPath(id int64) string      { return fmt.Sprintf("/api/v1/orders/%d", id) }
func transitionPath(id int64) string { return fmt.Sprintf("/api/v1/admin/orders/%d/transition", id) }

// decodeJSONBody decodes a success response body into v and closes it.
func decodeJSONBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

// TestAuthFlow walks the real credential path: register, then log in with the
// same password, then reach a protected endpoint with the issued token.
func TestAuthFlow(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Kofi Boateng", "email": "kofi@example.com", "phone": "0240000000", "password": "supersecret",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()

	resp = h.request(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "kofi@example.com", "password": "supersecret",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}
	var login authResponse
	decodeJSONBody(t, resp, &login)
	if login.AccessToken == "" {
		t.Fatal("login returned no access token")
	}

	resp = h.request(t, http.MethodGet, "/api/v1/me/loyalty", login.AccessToken, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("protected endpoint status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestLogin_WrongPasswordIsUnauthorized(t *testing.T) {
	h := newHarness(t)
	h.request(t, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Kofi", "email": "kofi2@example.com", "phone": "0240000000", "password": "supersecret",
	}).Body.Close()

	resp := h.request(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "kofi2@example.com", "password": "wrongpassword",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if code := decodeError(t, resp); code != codeInvalidCredentials {
		t.Errorf("code = %q, want %q", code, codeInvalidCredentials)
	}
}

func TestProtectedEndpoint_NoTokenIsUnauthorized(t *testing.T) {
	h := newHarness(t)
	resp := h.request(t, http.MethodGet, "/api/v1/me/loyalty", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestGetOrder_OwnershipReturns404 asserts the security rule that another user's
// order is indistinguishable from a missing one — a 404, never a 403 (so the API
// never leaks that the order exists).
func TestGetOrder_OwnershipReturns404(t *testing.T) {
	h := newHarness(t)
	owner := h.createUser(t, "owner@example.com", "")
	other := h.createUser(t, "other@example.com", "")
	order := h.seedOrder(t, owner, "CMUG-OWN-1")

	resp := h.request(t, http.MethodGet, orderPath(order.ID), h.token(t, owner), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner read status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	resp = h.request(t, http.MethodGet, orderPath(order.ID), h.token(t, other), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("non-owner read status = %d, want 404", resp.StatusCode)
	}
	if code := decodeError(t, resp); code != codeNotFound {
		t.Errorf("code = %q, want %q", code, codeNotFound)
	}
}

// TestStaffCannotMarkPaid asserts the core money rule at the HTTP boundary: even
// a staff member with a valid token cannot move an order to paid by hand. Only
// the payment webhook can.
func TestStaffCannotMarkPaid(t *testing.T) {
	h := newHarness(t)
	customer := h.createUser(t, "cust@example.com", "")
	staff := h.createUser(t, "staff@example.com", "staff")
	order := h.seedOrder(t, customer, "CMUG-STAFF-1")

	resp := h.request(t, http.MethodPost, transitionPath(order.ID), h.token(t, staff), map[string]string{"to": "paid"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	if code := decodeError(t, resp); code != codeForbidden {
		t.Errorf("code = %q, want %q", code, codeForbidden)
	}
	got, err := h.store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if got.Status != "pending_payment" {
		t.Errorf("status = %s, want pending_payment (unchanged)", got.Status)
	}
}

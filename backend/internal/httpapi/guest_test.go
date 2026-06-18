package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

// TestGuestSession_MintsUsableSession covers the happy S4 path: POST /auth/guest
// with a name and phone returns a normal session (access token + refresh cookie),
// and that token reaches a protected customer endpoint — proving a guest behaves
// like any logged-in customer downstream.
func TestGuestSession_MintsUsableSession(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodPost, "/api/v1/auth/guest", "", map[string]string{
		"name": "Akosua Frimpong", "phone": "0241234567",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("guest status = %d, want 201", resp.StatusCode)
	}
	var guest authResponse
	decodeJSONBody(t, resp, &guest)
	if guest.AccessToken == "" {
		t.Fatal("guest session returned no access token")
	}
	if guest.User.Name != "Akosua Frimpong" {
		t.Errorf("guest name = %q, want %q", guest.User.Name, "Akosua Frimpong")
	}
	if guest.User.Role != "customer" {
		t.Errorf("guest role = %q, want customer", guest.User.Role)
	}

	resp = h.request(t, http.MethodGet, "/api/v1/me/loyalty", guest.AccessToken, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("protected endpoint status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestGuestSession_AnonymousAllowed: an empty body is valid — a fully anonymous
// guest is created and labelled "Guest".
func TestGuestSession_AnonymousAllowed(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodPost, "/api/v1/auth/guest", "", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("guest status = %d, want 201", resp.StatusCode)
	}
	var guest authResponse
	decodeJSONBody(t, resp, &guest)
	if guest.User.Name != "Guest" {
		t.Errorf("anonymous guest name = %q, want %q", guest.User.Name, "Guest")
	}
}

// TestGuestSession_CannotLogIn proves the passwordless guarantee: the guest's
// synthetic email is real and unique, but its password hash is an unusable random
// secret, so a login attempt against that email always fails with the standard
// invalid-credentials error — there is no way to authenticate as a guest.
func TestGuestSession_CannotLogIn(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodPost, "/api/v1/auth/guest", "", map[string]string{"name": "Yaw"})
	var guest authResponse
	decodeJSONBody(t, resp, &guest)
	if !strings.HasSuffix(guest.User.Email, "@"+guestEmailDomain) {
		t.Fatalf("guest email = %q, want a @%s address", guest.User.Email, guestEmailDomain)
	}

	resp = h.request(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": guest.User.Email, "password": "anything-at-all",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login as guest status = %d, want 401", resp.StatusCode)
	}
	if code := decodeError(t, resp); code != codeInvalidCredentials {
		t.Errorf("code = %q, want %q", code, codeInvalidCredentials)
	}
}

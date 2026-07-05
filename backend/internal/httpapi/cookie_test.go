package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"coffeemug/backend/internal/config"
)

// refreshCookie extracts the refresh cookie from a recorded response.
func refreshCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			return c
		}
	}
	t.Fatalf("no %s cookie in response", refreshCookieName)
	return nil
}

// The refresh cookie must flip SameSite and Secure together with the
// COOKIE_SECURE config: Lax/!Secure for same-site local dev, None/Secure for a
// cross-origin deployment (a browser rejects SameSite=None without Secure, and
// silently drops a Lax cookie on cross-site requests).
func TestRefreshCookieAttributes(t *testing.T) {
	cases := []struct {
		name         string
		cookieSecure bool
		wantSameSite http.SameSite
	}{
		{"dev same-site", false, http.SameSiteLaxMode},
		{"prod cross-origin", true, http.SameSiteNoneMode},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{cfg: &config.Config{CookieSecure: tc.cookieSecure}}

			rec := httptest.NewRecorder()
			s.setRefreshCookie(rec, "tok", time.Now().Add(time.Hour))
			c := refreshCookie(t, rec)
			if c.Secure != tc.cookieSecure {
				t.Errorf("set: Secure = %v, want %v", c.Secure, tc.cookieSecure)
			}
			if c.SameSite != tc.wantSameSite {
				t.Errorf("set: SameSite = %v, want %v", c.SameSite, tc.wantSameSite)
			}
			if !c.HttpOnly {
				t.Error("set: HttpOnly = false, want true")
			}

			rec = httptest.NewRecorder()
			s.clearRefreshCookie(rec)
			c = refreshCookie(t, rec)
			if c.Secure != tc.cookieSecure {
				t.Errorf("clear: Secure = %v, want %v", c.Secure, tc.cookieSecure)
			}
			if c.SameSite != tc.wantSameSite {
				t.Errorf("clear: SameSite = %v, want %v", c.SameSite, tc.wantSameSite)
			}
			if c.MaxAge >= 0 {
				t.Errorf("clear: MaxAge = %d, want negative (delete)", c.MaxAge)
			}
		})
	}
}

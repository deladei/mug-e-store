package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSecurityHeaders asserts the defensive headers are present on every
// response and that the middleware still delegates to the wrapped handler. The
// middleware reads no Server state, so a zero Server is enough to exercise it
// without a database.
func TestSecurityHeaders(t *testing.T) {
	var reached bool
	h := (&Server{}).securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil))

	if !reached {
		t.Fatal("securityHeaders did not call the next handler")
	}
	want := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Content-Security-Policy":      "default-src 'none'; frame-ancestors 'none'",
		"Cross-Origin-Resource-Policy": "same-site",
		"Strict-Transport-Security":    "max-age=31536000; includeSubDomains",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("header %s = %q, want %q", k, got, v)
		}
	}
}

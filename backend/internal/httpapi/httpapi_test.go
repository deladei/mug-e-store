package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"coffeemug/backend/internal/auth"
	"coffeemug/backend/internal/config"
	"coffeemug/backend/internal/paystack"
	"coffeemug/backend/internal/sse"
	"coffeemug/backend/internal/store"
)

// HTTP-level tests exercise the real router, middleware and handlers against a
// real store, with Paystack faked by a local httptest server. They reuse the
// store suite's TEST_DATABASE_URL and skip when it is unset. This suite shares
// the one test database with the store suite and truncates it, so run the whole
// tree with `go test -p 1 ./...` to serialize the package binaries.

const testPaystackSecret = "sk_test_harness_secret_0123456789"

type harness struct {
	srv    *httptest.Server
	store  *store.Store
	tokens *auth.TokenManager

	// Controls for the fake Paystack verify endpoint; a test sets these before
	// firing a webhook to drive each of the payment gates.
	verifyStatus   string
	verifyAmount   int64
	verifyCurrency string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run httpapi tests against Postgres")
	}
	st, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	const truncate = `TRUNCATE
		loyalty_ledger, order_events, order_lines, orders,
		cart_lines, carts, item_variants, items, categories,
		password_reset_tokens, refresh_tokens, users RESTART IDENTITY CASCADE`
	if _, err := st.DB().ExecContext(context.Background(), truncate); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	h := &harness{store: st, verifyStatus: "success", verifyCurrency: "GHS"}

	// Fake Paystack: only the verify endpoint matters for these tests.
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  true,
			"message": "ok",
			"data": map[string]any{
				"status":    h.verifyStatus,
				"amount":    h.verifyAmount,
				"currency":  h.verifyCurrency,
				"reference": "ref",
			},
		})
	}))
	t.Cleanup(fake.Close)

	cfg := &config.Config{
		JWTSecret:          "jwt-test-secret-abcdefghijklmnop",
		PaystackSecretKey:  testPaystackSecret,
		PaystackBaseURL:    fake.URL,
		Port:               "0",
		DeliveryFeePesewas: 1000,
		FrontendOrigin:     "http://localhost:3000",
	}
	h.tokens = auth.NewTokenManager(cfg.JWTSecret)
	ps := paystack.NewClient(testPaystackSecret, fake.URL)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer(cfg, st, h.tokens, ps, sse.NewBroker(), logger)

	h.srv = httptest.NewServer(server.Handler())
	t.Cleanup(h.srv.Close)
	return h
}

// createUser inserts a user with the given role (empty = customer default).
func (h *harness) createUser(t *testing.T, email, role string) *store.User {
	t.Helper()
	u := &store.User{Name: "Ama Owusu", Email: email, Phone: "0240000000", PasswordHash: "x", Role: role}
	if err := h.store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// token mints an access token for a user.
func (h *harness) token(t *testing.T, u *store.User) string {
	t.Helper()
	tok, err := h.tokens.GenerateAccessToken(u.ID, u.Role)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

// seedOrder creates a pending_payment order for a user with a known reference and
// total, the minimal setup for the order/webhook tests.
func (h *harness) seedOrder(t *testing.T, u *store.User, reference string) *store.Order {
	t.Helper()
	ctx := context.Background()
	cat := &store.Category{Name: "Coffee", SortOrder: 1}
	if err := h.store.CreateCategory(ctx, cat); err != nil {
		t.Fatalf("category: %v", err)
	}
	item := &store.Item{CategoryID: cat.ID, Name: "Latte"}
	if err := h.store.CreateItem(ctx, item); err != nil {
		t.Fatalf("item: %v", err)
	}
	v := &store.Variant{ItemID: item.ID, Name: "Regular", PricePesewas: 2000, SortOrder: 1}
	if err := h.store.CreateVariant(ctx, v); err != nil {
		t.Fatalf("variant: %v", err)
	}
	if _, err := h.store.AddCartItem(ctx, u.ID, v.ID, 1); err != nil {
		t.Fatalf("add to cart: %v", err)
	}
	order, err := h.store.Checkout(ctx, store.CheckoutParams{UserID: u.ID, Fulfilment: "pickup"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if err := h.store.SetPaystackReference(ctx, order.ID, reference); err != nil {
		t.Fatalf("set reference: %v", err)
	}
	return order
}

// request fires an HTTP request to the test server and returns the response. body
// may be nil; token, if non-empty, is sent as a Bearer header.
func (h *harness) request(t *testing.T, method, path, token string, body any) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, h.srv.URL+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := h.srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

// decodeError reads the standard error envelope's code from a response.
func decodeError(t *testing.T, resp *http.Response) string {
	t.Helper()
	var env errEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return env.Error.Code
}

// signPaystack produces a valid x-paystack-signature for a webhook body.
func signPaystack(secret string, body []byte) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Package httpapi is the HTTP layer: it translates requests into store/domain
// calls and back into JSON. It holds no business rules — those live in
// internal/domain and internal/store. Routing uses the Go 1.22 stdlib
// method+path mux; there is no web framework.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"coffeemug/backend/internal/auth"
	"coffeemug/backend/internal/config"
	"coffeemug/backend/internal/paystack"
	"coffeemug/backend/internal/sse"
	"coffeemug/backend/internal/store"
)

// Server bundles the dependencies the handlers need. It is constructed once in
// cmd/api and is safe for concurrent use.
type Server struct {
	cfg      *config.Config
	store    *store.Store
	tokens   *auth.TokenManager
	paystack *paystack.Client
	broker   *sse.Broker
	logger   *slog.Logger
	limiter  *rateLimiter
}

// NewServer wires the dependencies together.
func NewServer(cfg *config.Config, st *store.Store, tm *auth.TokenManager, ps *paystack.Client, broker *sse.Broker, logger *slog.Logger) *Server {
	return &Server{
		cfg:      cfg,
		store:    st,
		tokens:   tm,
		paystack: ps,
		broker:   broker,
		logger:   logger,
		limiter:  newRateLimiter(10, time.Minute), // 10 auth attempts/min per IP
	}
}

// Handler builds the full route table and wraps it in the global middleware
// chain (panic recovery → security headers → request logging → CORS). The
// returned handler is what cmd/api serves.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health.
	mux.HandleFunc("GET /api/v1/healthz", s.handleHealthz)

	// Auth (public, rate-limited on the credential endpoints).
	mux.Handle("POST /api/v1/auth/register", s.rateLimit(http.HandlerFunc(s.handleRegister)))
	mux.Handle("POST /api/v1/auth/login", s.rateLimit(http.HandlerFunc(s.handleLogin)))
	mux.Handle("POST /api/v1/auth/guest", s.rateLimit(http.HandlerFunc(s.handleGuestSession)))
	mux.HandleFunc("POST /api/v1/auth/refresh", s.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.Handle("POST /api/v1/auth/password-reset/request", s.rateLimit(http.HandlerFunc(s.handlePasswordResetRequest)))
	mux.Handle("POST /api/v1/auth/password-reset/confirm", s.rateLimit(http.HandlerFunc(s.handlePasswordResetConfirm)))

	// Catalog (public reads).
	mux.HandleFunc("GET /api/v1/categories", s.handleListCategories)
	mux.HandleFunc("GET /api/v1/items", s.handleListItems)
	mux.HandleFunc("GET /api/v1/items/{id}", s.handleGetItem)

	// Cart (Bearer).
	mux.Handle("GET /api/v1/cart", s.requireAuth(http.HandlerFunc(s.handleGetCart)))
	mux.Handle("POST /api/v1/cart/items", s.requireAuth(http.HandlerFunc(s.handleAddCartItem)))
	mux.Handle("PATCH /api/v1/cart/items/{lineId}", s.requireAuth(http.HandlerFunc(s.handleUpdateCartItem)))
	mux.Handle("DELETE /api/v1/cart/items/{lineId}", s.requireAuth(http.HandlerFunc(s.handleRemoveCartItem)))

	// Checkout + customer order views (Bearer).
	mux.Handle("POST /api/v1/checkout", s.requireAuth(http.HandlerFunc(s.handleCheckout)))
	mux.Handle("GET /api/v1/me/orders", s.requireAuth(http.HandlerFunc(s.handleListMyOrders)))
	mux.Handle("GET /api/v1/me/loyalty", s.requireAuth(http.HandlerFunc(s.handleLoyalty)))
	mux.Handle("GET /api/v1/orders/{id}", s.requireAuth(http.HandlerFunc(s.handleGetOrder)))
	// SSE accepts the token in the query string (EventSource cannot set headers).
	mux.Handle("GET /api/v1/orders/{id}/events", s.requireAuthQuery(http.HandlerFunc(s.handleOrderEvents)))

	// Webhook (Paystack only; authenticated by signature, not a token).
	mux.HandleFunc("POST /api/v1/webhooks/paystack", s.handlePaystackWebhook)

	// Staff/admin: order queue + advance + availability.
	mux.Handle("GET /api/v1/admin/orders", s.requireRole(http.HandlerFunc(s.handleAdminListOrders), "staff", "admin"))
	mux.Handle("GET /api/v1/admin/orders/{id}/history", s.requireRole(http.HandlerFunc(s.handleAdminOrderHistory), "staff", "admin"))
	mux.Handle("POST /api/v1/admin/orders/{id}/transition", s.requireRole(http.HandlerFunc(s.handleAdminTransition), "staff", "admin"))
	mux.Handle("PATCH /api/v1/admin/items/{id}/availability", s.requireRole(http.HandlerFunc(s.handleSetAvailability), "staff", "admin"))

	// Admin-only: menu management.
	mux.Handle("POST /api/v1/admin/categories", s.requireRole(http.HandlerFunc(s.handleCreateCategory), "admin"))
	mux.Handle("PATCH /api/v1/admin/categories/{id}", s.requireRole(http.HandlerFunc(s.handleUpdateCategory), "admin"))
	mux.Handle("DELETE /api/v1/admin/categories/{id}", s.requireRole(http.HandlerFunc(s.handleDeleteCategory), "admin"))
	mux.Handle("POST /api/v1/admin/items", s.requireRole(http.HandlerFunc(s.handleCreateItem), "admin"))
	mux.Handle("PATCH /api/v1/admin/items/{id}", s.requireRole(http.HandlerFunc(s.handleUpdateItem), "admin"))
	mux.Handle("DELETE /api/v1/admin/items/{id}", s.requireRole(http.HandlerFunc(s.handleDeleteItem), "admin"))
	mux.Handle("POST /api/v1/admin/items/{id}/variants", s.requireRole(http.HandlerFunc(s.handleCreateVariant), "admin"))
	mux.Handle("DELETE /api/v1/admin/variants/{id}", s.requireRole(http.HandlerFunc(s.handleDeleteVariant), "admin"))

	// Admin-only: business reports (revenue is financial data → admin, not staff).
	mux.Handle("GET /api/v1/admin/reports/summary", s.requireRole(http.HandlerFunc(s.handleReportSummary), "admin"))

	return s.recoverPanic(s.securityHeaders(s.logRequests(s.cors(mux))))
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DB().PingContext(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, codeInternal, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// pathInt parses a {name} path value as int64, writing a 404 on failure (a
// non-numeric id can never identify a real row).
func (s *Server) pathInt(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	v, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, codeNotFound, "not found")
		return 0, false
	}
	return v, true
}

// ctxKey is the unexported type for request-context keys.
type ctxKey int

const userCtxKey ctxKey = iota

// authUser is the authenticated principal extracted from a valid access token.
type authUser struct {
	ID   int64
	Role string
}

// userFrom returns the authenticated user placed in the context by requireAuth.
func userFrom(ctx context.Context) (authUser, bool) {
	u, ok := ctx.Value(userCtxKey).(authUser)
	return u, ok
}

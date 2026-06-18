# Coffee Mug Shop — Technical Requirements Document (TRD)

**Scope:** Backend service only. The Next.js frontend is owned by other team members; this document defines the server, its contracts, and its guarantees.
**Status:** Describes the system as built. The backend compiles, runs, and passes its test suite. This is documentation of reality, not a forward proposal.
**Companion documents:** PRD (`coffee-mug-shop-prd.md`), Database Schema (`coffee-mug-shop-database-schema.md`), User Journey (`coffee-mug-shop-user-journey.md`), API Consumption Brief (`coffee-mug-shop-api-consumption-brief.md`), Implementation Plan (`coffee-mug-shop-implementation-plan.md`).
**Version:** 1.0

---

## 1. Purpose and boundaries

The backend is a single Go service that exposes a JSON/HTTP API for a one-location coffee shop. It owns: identity and sessions, the menu catalog, the server-side cart, checkout and payment verification, the order lifecycle, real-time order status, and the admin/staff control surface.

It does **not** own, and will never contain: any UI rendering, any rider tracking or maps, any in-app card data capture (Paystack's hosted page handles all card entry), multi-shop or franchise logic, or SMS/email delivery. These are out of scope by design, not by omission. The single most important boundary: **the backend never accepts a client's claim that a payment succeeded** — payment truth comes only from a signature-verified Paystack webhook plus a server-to-server verify call.

---

## 2. Architecture

### 2.1 Shape

A single Go binary, layered so that each layer may only call the one below it:

```
cmd/api                     process entry point, graceful shutdown
  └── internal/httpapi       HTTP: routing, middleware, request/response, auth gate
        └── internal/store    the only package that speaks SQL
              └── internal/domain   pure business rules, zero dependencies
```

Supporting packages, each with a single responsibility:

- `internal/config` — loads all configuration from environment variables; refuses to start if a required secret is missing.
- `internal/auth` — password hashing, JWT access tokens, refresh-token generation and hashing.
- `internal/paystack` — the Paystack client (initialize, verify) and webhook signature verification.
- `internal/sse` — an in-process publish/subscribe broker for order-status events.

The rule that keeps this honest: **if a business rule about orders or money lives anywhere except `internal/domain`, that is a bug.** Handlers translate HTTP to function calls; the store translates function calls to SQL; the domain decides what is allowed. A reviewer can read `internal/domain/order.go` alone and understand the entire order lifecycle.

### 2.2 Why a monolith and not microservices

One shop, one team, one semester. Splitting auth, catalog, and orders into separate services would add network calls, deployment surface, and distributed-transaction problems to solve nothing this project has. The layering above already gives the separation of concerns that matters; process boundaries would only add cost. This is a deliberate decision recorded so it can be defended, not an accident of scope.

### 2.3 Why no Redis / message queue for real-time

Real-time order tracking is delivered over Server-Sent Events from an **in-process** broker (`internal/sse`). One server process serves both the writes (a staff transition) and the reads (a customer's open SSE stream), so they already share memory — an external broker would be infrastructure with no job. The cost of this choice is explicit and accepted: **it does not survive horizontal scaling.** If a second instance is ever run, a customer connected to instance A will not receive an event published on instance B. The migration path (swap the in-process broker for Redis pub/sub behind the same `Broker` interface) is small and is recorded in the Implementation Plan. For one shop it will never be hit.

---

## 3. Technology stack

| Concern | Choice | Version | Rationale |
|---|---|---|---|
| Language | Go | 1.22+ | Standard-library HTTP routing with method+path patterns (1.22 feature) removes the need for a web framework. Single static binary deploys trivially. |
| HTTP routing | `net/http` stdlib | — | `mux.HandleFunc("POST /api/v1/...")` and `r.PathValue("id")` cover every route here. No Gin/Echo/Chi dependency to justify. |
| Database | PostgreSQL | 16 | Relational integrity, `CHECK` constraints, transactions, and `FOR UPDATE` row locks are all load-bearing in this design (see §6). |
| DB driver | `github.com/lib/pq` | 1.10.9 | `database/sql`-compatible Postgres driver; no ORM. |
| Auth tokens | `github.com/golang-jwt/jwt/v5` | 5.2.1 | HS256 access tokens. |
| Password hashing | `golang.org/x/crypto/bcrypt` | 0.23 | bcrypt cost 12. |

No ORM is used. Queries are hand-written SQL in `internal/store`, which keeps the exact query — and its cost — visible at the point of use. This is a defensible choice for a system this size; the trade-off (more boilerplate, no automatic migrations from structs) is accepted in exchange for transparency.

---

## 4. Functional surface (API)

Full request/response detail lives in the API Consumption Brief. This is the index of what exists, grouped by access level. Base path: `/api/v1`.

**Public (no token)**
- `POST /auth/register`, `POST /auth/login` — rate limited (see §5.4)
- `POST /auth/refresh`, `POST /auth/logout`
- `GET /categories`, `GET /items`, `GET /items/{id}`
- `POST /webhooks/paystack` — authenticated by signature, not by token
- `GET /healthz`

**Customer (valid access token)**
- `GET /cart`, `POST /cart/items`, `PATCH /cart/items/{lineId}`, `DELETE /cart/items/{lineId}`
- `POST /checkout`
- `GET /me/orders`, `GET /me/loyalty`
- `GET /orders/{id}`, `GET /orders/{id}/events` (SSE)

**Staff and Admin**
- `GET /admin/orders`, `GET /admin/orders/{id}/history`
- `POST /admin/orders/{id}/transition`
- `PATCH /admin/items/{id}/availability`

**Admin only**
- `POST/PATCH/DELETE /admin/categories`, `POST/PATCH/DELETE /admin/items`
- `POST /admin/items/{id}/variants`, `DELETE /admin/variants/{id}`

---

## 5. Non-functional requirements

### 5.1 Money

All monetary values are **integer pesewas** (1 GHS = 100 pesewas), in `BIGINT` columns and `int64` in Go. Floating-point money is never used anywhere — not in the database, not in transport, not in calculation. The frontend formats for display; the backend only ever moves integers. The amount sent to Paystack is the same integer stored on the order, and the webhook's verified amount is compared to it exactly.

### 5.2 Payment integrity (the core requirement)

An order becomes `paid` only when **all four** of these hold, in order:

1. The webhook request body carries a valid `x-paystack-signature` — HMAC-SHA512 of the raw body keyed with the secret key.
2. A server-to-server `GET /transaction/verify/{reference}` call to Paystack returns `status: success`.
3. The verified amount equals the order's stored `total_pesewas` **and** the currency is `GHS`.
4. The order is in a state from which `paid` is reachable.

A client calling any endpoint cannot move an order to `paid`. Staff cannot move an order to `paid` by hand (the transition endpoint explicitly forbids it). This is the requirement the whole design exists to protect.

### 5.3 Idempotency

Two independent idempotency mechanisms:

- **Checkout:** the client may send an `idempotency_key`. It is stored `UNIQUE` on the order. A retried checkout with the same key returns `409 duplicate_order` instead of creating a second order.
- **Payment:** Paystack retries webhooks. The `paystack_reference` is `UNIQUE`, and the state transition treats "move to the status it is already in" as a successful no-op. A webhook delivered five times marks the order paid once and never errors.

### 5.4 Security

- Passwords: bcrypt cost 12. Plaintext is never logged or stored.
- Access tokens: JWT HS256, 15-minute expiry, carrying `uid` and `role` claims only. The signing method is asserted on parse (a token claiming `alg: none` is rejected).
- Refresh tokens: 32 bytes of CSPRNG randomness. Only the **SHA-256 hash** is stored, so a database leak does not yield usable sessions. Tokens are single-use: refreshing consumes (deletes) the old token and issues a new one (rotation). Delivered as an `httpOnly` cookie scoped to `/api/v1/auth`.
- Account enumeration: login and registration return identical errors for "unknown email" and "wrong password," and registration does not reveal whether an email exists beyond a generic `email_taken` on the dedicated path.
- Authorization: ownership checks on order endpoints return `404`, not `403`, so a customer cannot probe which order IDs exist.
- Rate limiting: an in-memory per-IP token bucket (10 requests/minute, burst 10) on register and login only.
- CORS: credentials allowed for exactly one configured origin (`FRONTEND_ORIGIN`); not a wildcard.
- Request bodies are capped at 1 MB; JSON decoding rejects unknown fields.

### 5.5 Reliability

- Checkout is a single database transaction: read cart at current prices → insert order → snapshot lines → clear cart. Any failure rolls the whole thing back; there is no half-created order.
- State transitions take a `FOR UPDATE` row lock so two concurrent transitions on one order cannot race.
- The HTTP server has a read-header timeout and idle timeout but **no write timeout**, because SSE connections are intentionally long-lived. Shutdown is graceful (in-flight requests drain for up to 10 seconds).
- A panic in any handler is recovered and returned as a `500` with the standard error envelope; the process does not crash.

### 5.6 Observability

Structured logging via `log/slog`: every request logs method, path, and duration; every recovered panic and every payment anomaly (bad signature, amount mismatch, unknown reference, transition conflict) logs at warn/error with the order or reference attached.

---

## 6. Data integrity rules enforced server-side

These are guarantees, not conventions — they hold regardless of what any client sends.

1. **Order history is immutable.** `order_lines` snapshot the item name, variant name, and unit price at checkout. Editing the menu afterward never changes a past order's contents or totals.
2. **The order lifecycle is a finite state machine** with one source of truth (`internal/domain`). Illegal jumps (e.g. `pending_payment` → `completed`) are rejected. The machine is fulfilment-aware: a pickup order can never enter `out_for_delivery`; a delivery order can never reach `completed` without first going `out_for_delivery`.
3. **Loyalty is an append-only ledger.** Balance is always `SUM(delta)`; no mutable balance column exists, so the points total cannot drift out of sync with its history. Points are earned (1 per GHS 1 of subtotal) **only** on the `completed` transition — an order cancelled after payment earns nothing.
4. **Cancellation is only possible early.** An order may be cancelled from `pending_payment` or `paid`, never once preparation has begun.
5. **Availability is enforced at the backend.** Adding an unavailable item to the cart is rejected server-side; checkout re-checks availability inside the transaction. Greying out a button in the UI is a convenience, not the control.

---

## 7. Configuration (runtime requirements)

All configuration is environment-only (12-factor). The service refuses to start if `DATABASE_URL`, `JWT_SECRET`, or `PAYSTACK_SECRET_KEY` is absent.

| Variable | Required | Default | Meaning |
|---|---|---|---|
| `DATABASE_URL` | yes | — | Postgres connection string |
| `JWT_SECRET` | yes | — | HS256 signing secret (use a long random string) |
| `PAYSTACK_SECRET_KEY` | yes | — | Paystack secret key (`sk_test_…` in development) |
| `PAYSTACK_BASE_URL` | no | `https://api.paystack.co` | Overridable for testing against a mock |
| `PORT` | no | `8080` | Listen port |
| `DELIVERY_FEE_PESEWAS` | no | `1000` | Flat delivery fee (GHS 10) |
| `FRONTEND_ORIGIN` | no | `http://localhost:3000` | The single allowed CORS origin |

---

## 8. Testing requirements

- **Domain:** the state machine is exhaustively unit-tested — every legal and illegal transition for both fulfilment types, plus terminality. This is the highest-value test surface because it encodes the rules everything else depends on.
- **Payments:** webhook signature verification is unit-tested against a forged signature, a valid signature, a signature from the wrong secret, and a tampered body — all without any network.
- **What is deliberately not testable in this sandbox, and is therefore the developer's responsibility:** a true end-to-end Paystack round trip (real keys, a public callback URL via a tunnel). This is named explicitly in the Implementation Plan rather than faked.

---

## 9. Known limitations (stated, not hidden)

- Real-time delivery is single-instance only (see §2.3).
- The rate limiter is in-memory and per-instance; it resets on restart and is per-process.
- Loyalty **redemption at checkout** is intentionally not implemented — it is reserved as the developer's own task and is fully specified in the PRD and Implementation Plan. Earning is implemented; spending is not.
- There is no automated cleanup of expired refresh tokens yet (they are simply rejected when used after expiry). A periodic delete is a trivial future addition.

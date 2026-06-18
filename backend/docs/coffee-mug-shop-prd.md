# Coffee Mug Shop — Product Requirements Document

**Version:** 1.0 (Draft)
**Author:** Backend — Alan Kobby Dei (Walker) · Frontend — [names]
**Status:** For team review
**Course context:** Semester project, ecommerce application
**Stack:** Go (backend API) · Next.js (frontend) · PostgreSQL · Paystack (test mode)

---

## 1. Overview

Coffee Mug Shop is an online ordering platform for a single coffee shop. Customers browse the menu, place orders for pickup or delivery, pay via Paystack, and track their order status in real time. Shop staff manage the menu and process incoming orders through an admin dashboard. Repeat customers earn loyalty points on completed orders.

This is a **single-tenant** system: one shop, one menu, one admin team. This constraint simplifies almost every backend decision (no tenant isolation, no per-shop RLS-style scoping, no vendor onboarding) and should be stated explicitly so nobody accidentally designs for multi-vendor.

### Problem statement

Small coffee shops in Ghana take orders over WhatsApp and phone calls. Orders get lost, there's no payment confirmation before preparation starts, and there's no record of customer history. Coffee Mug Shop replaces this with a structured order pipeline: browse → order → pay → prepare → fulfil.

### Why this matters for the project grade

The system demonstrates the full ecommerce loop: catalog, cart, checkout, payment integration with a real gateway, order lifecycle management, and role-based access. That is the academic point. Everything in this document is in service of demonstrating that loop working end-to-end, reliably, in a demo.

---

## 2. Goals and Non-Goals

### Goals

1. A customer can complete a full order (browse → cart → checkout → pay → receive status updates) without staff intervention.
2. Staff can manage the menu and move orders through their lifecycle from a dashboard.
3. Payments are verified server-side via Paystack webhooks — the system never trusts the client to confirm payment.
4. Order status changes propagate to the customer's screen without a manual refresh.
5. Completed orders award loyalty points that can be redeemed as a discount.

### Non-Goals (explicitly out of scope)

- Multi-shop / multi-vendor support
- Delivery rider assignment, tracking, or logistics (delivery is "staff marks it delivered")
- Geocoding, delivery zones, or distance-based pricing — delivery fee is **flat**
- Native mobile apps (the Next.js frontend should be responsive; that is sufficient)
- Inventory/stock management beyond a simple "available / unavailable" toggle per item
- Refunds through the system (handled manually outside the app; the order can be marked `cancelled`)
- Email/SMS notifications (status updates are in-app only)

Non-goals are as load-bearing as goals. Every one of these was cut deliberately — if any teammate wants one back in, they argue against the deadline, not against the document.

---

## 3. Users and Roles

| Role | Description | Access |
|---|---|---|
| **Customer** | Orders coffee. May be a guest browser, but must register to check out. | Public catalog; own cart, orders, profile, points |
| **Staff** | Barista/cashier. Works the order queue. | Order queue, status transitions, item availability toggle |
| **Admin** | Shop owner/manager. Everything staff can do, plus menu CRUD and basic reports. | Full dashboard |

Three roles, one `role` column on the users table. Do not build a permissions framework — a middleware check (`requireRole("staff", "admin")`) is all this project needs.

---

## 4. Scope and Phasing

This is the most important section of the document. The features below are split into a **Phase 1 MVP** that constitutes a complete, gradable, demoable product, and a **Phase 2** that is attempted only after Phase 1 is deployed and stable.

### Phase 1 — MVP (the grade)

| # | Feature | Notes |
|---|---|---|
| F1 | Auth: register, login, JWT sessions | Email + password. No OAuth, no password reset flow (stretch). |
| F2 | Menu catalog: categories, items, variants | e.g. Latte → Small/Medium/Large with price deltas |
| F3 | Cart | Server-side cart tied to user |
| F4 | Checkout: pickup OR delivery (flat fee) + Paystack payment | Paystack test mode, GHS |
| F5 | Payment verification via webhook | Source of truth for `paid` status |
| F6 | Order lifecycle + staff order queue | `pending_payment → paid → preparing → ready → completed` (+ `out_for_delivery` for delivery orders, + `cancelled`) |
| F7 | Real-time order status for customer | Server-Sent Events (SSE) |
| F8 | Admin: menu CRUD, item availability toggle | |
| F9 | Customer order history | |

### Phase 2 — Stretch (the demo flex)

| # | Feature | Notes |
|---|---|---|
| S1 | Loyalty points: earn on completion, redeem at checkout | Append-only ledger (see §8) |
| S2 | Basic admin reports | Orders/day, revenue/day — two SQL queries and a chart |
| S3 | Password reset | Requires email sending; that's why it's stretch |
| S4 | Guest checkout | Only if frontend team has spare capacity |

**Rule:** no Phase 2 work starts until every Phase 1 feature passes the demo script in §12. Loyalty points were requested as in-scope; they are deliberately placed in Phase 2 because they are the only feature whose absence does not break the core ecommerce loop, and they are the easiest to add late (one table, two write paths, one read path).

---
## 5. Functional Requirements (detail)

### F1 — Authentication

Registration requires name, email, phone, password. Passwords hashed with bcrypt (cost 10–12). Login returns a short-lived access JWT (15 min) and a refresh token (7 days, stored as an httpOnly cookie). The frontend never stores tokens in localStorage.

Acceptance criteria: a registered user can log in, receive a token, call a protected endpoint, refresh an expired access token, and log out (refresh token revoked server-side via a `token_version` or revocation table).

### F2 — Catalog

Items belong to one category (Espresso Drinks, Brewed, Pastries, etc.). An item has one or more **variants** (size/options) each with its own price. Prices are stored in **pesewas as integers** — never floats, never `numeric` parsed into float64. All money math is integer math.

Items have an `is_available` boolean. Unavailable items appear greyed out (frontend decision) but cannot be added to a cart (backend enforces).

### F3 — Cart

One active cart per user, server-side. Endpoints to add item-variant + quantity, update quantity, remove line, view cart. Cart lines snapshot nothing — prices are resolved at checkout time, not add-to-cart time. If a price changes between add and checkout, checkout recalculates and the frontend shows the new total. (Snapshotting at add-time is the alternative; it is more code for a problem a coffee shop does not have.)

### F4 + F5 — Checkout and Payment

1. Customer chooses fulfilment: `pickup` or `delivery`. Delivery requires a free-text address and phone, and adds a **flat fee** (configurable constant, e.g. GHS 10).
2. Backend creates an `order` in `pending_payment` with line items **snapshotted** (name, variant, unit price at that moment) — order lines are immutable historical records, unlike cart lines.
3. Backend calls Paystack `transaction/initialize` with the order total and a generated `reference`, returns the authorization URL to the frontend.
4. Customer pays on Paystack's hosted page (test cards).
5. **Paystack calls our webhook.** Backend verifies the `x-paystack-signature` HMAC, re-fetches the transaction via Paystack's verify endpoint, checks `amount` and `currency` match the order, and only then transitions the order to `paid`.
6. The frontend callback page polls `GET /orders/{id}` (or listens on SSE) until status is `paid`. The callback redirect is **UX only** — it never confirms payment.

**Idempotency:** the webhook handler must be idempotent (Paystack retries). Transitioning `paid → paid` is a no-op, not an error. Order creation should also accept an idempotency key from the client to survive double-clicks on "Place Order".

**Critical rule for the whole team:** client-side "payment success" signals are decorative. If the webhook hasn't fired, the order is not paid. This is the single most common way student projects (and real businesses) get this wrong.

### F6 — Order Lifecycle

```
pending_payment ──> paid ──> preparing ──> ready ──┬──> completed        (pickup)
        │                                          └──> out_for_delivery ──> completed   (delivery)
        └──> cancelled (allowed from pending_payment or paid only)
```

Transitions are enforced server-side with an explicit allowed-transitions map. Staff cannot jump `paid → completed`. Invalid transitions return `409 Conflict`.

The staff queue view lists orders grouped by status, newest `paid` first. Each status change is written to an `order_events` table (order_id, from_status, to_status, actor_id, timestamp) — this is your audit trail and it also makes the demo narrative easy ("here's the full history of this order").

### F7 — Real-time Status (SSE)

Customer order page subscribes to `GET /orders/{id}/events` (SSE). On any status change the backend pushes the new status. Staff dashboard can use simple 10-second polling — staff are looking at the screen anyway and polling is one line of frontend code.

**Why SSE and not WebSockets:** the data flow is strictly server→client, SSE is plain HTTP (no upgrade handshake, works through proxies, trivially testable with curl), auto-reconnects natively in the browser, and in Go it's just a flusher loop on the response writer. WebSockets buy you bidirectionality you don't need at the cost of connection-management code you'll have to debug during exam week. If a teammate pushes for WebSockets, the burden of proof is on them to name the client→server message that justifies it.

Implementation note: a single in-process pub/sub (a `map[orderID][]chan StatusEvent` guarded by a mutex, or a small broker goroutine) is sufficient. Do **not** introduce Redis for this — one server process, one shop.

### F8 — Admin Menu Management

CRUD for categories, items, variants. Image upload can be a URL field for MVP (frontend uses any hosted image); actual file upload to object storage is a nice-to-have, not a requirement.

### F9 — Order History

`GET /me/orders` paginated, newest first, with line items. Nothing clever.

### S1 — Loyalty (Phase 2)

Earn: 1 point per GHS 1 of the **subtotal** (not delivery fee), awarded when an order transitions to `completed` — not at payment, because cancelled-after-payment orders shouldn't earn.

Redeem: at checkout, customer applies points; 100 points = GHS 1 off (tune the ratio). Redemption creates a negative ledger entry at order creation; if the order is cancelled before completion, a compensating positive entry is written.

**Storage rule:** points live in an append-only `loyalty_ledger` (user_id, order_id, delta, reason, created_at). Balance = `SUM(delta)`. Never store a mutable `points_balance` column as the source of truth — a ledger can't get out of sync with itself, survives concurrent writes, and gives you statement history for free. (Cache the sum later if it's ever slow. It won't be.)

---

## 6. System Architecture

```
[Next.js frontend] ──HTTPS/JSON──> [Go API (single binary)] ──> [PostgreSQL]
        │                                   │
        │<────────── SSE ───────────────────┤
                                            │<── webhook ── [Paystack]
```

One Go binary. Layered, not microserviced:

- `cmd/api/main.go` — wiring, config, server
- `internal/http/` — handlers, middleware (auth, logging, recovery, CORS), routes
- `internal/service/` — business logic (order state machine, checkout, loyalty)
- `internal/store/` — Postgres access
- `internal/paystack/` — gateway client (initialize, verify, signature check)
- `internal/sse/` — broker

**Recommended libraries:** Go 1.22+ stdlib `net/http` routing (method+path patterns make chi optional now — pick chi only if the team already knows it), `pgx/v5` for Postgres, `sqlc` if you want generated type-safe queries (recommended — it forces you to write real SQL, which is also good viva material), `golang-jwt/jwt/v5`. Resist ORMs; GORM will save you two hours in week one and cost you ten in week eight.

**Config** via environment variables only: `DATABASE_URL`, `PAYSTACK_SECRET_KEY`, `JWT_SECRET`, `PORT`, `DELIVERY_FEE_PESEWAS`, `FRONTEND_ORIGIN` (for CORS). A `.env.example` file is part of the deliverable.

**Migrations:** `golang-migrate` or `goose`, committed to the repo. The grader should be able to run `make migrate && make run`.

---

## 7. API Surface (v1)

All endpoints under `/api/v1`. JSON in/out. Errors follow one envelope: `{ "error": { "code": "string", "message": "string" } }`.

**Auth**
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`

**Catalog (public)**
- `GET /categories`
- `GET /items?category=` (only available items for non-staff)
- `GET /items/{id}`

**Cart (customer)**
- `GET /cart`
- `POST /cart/items` `{item_variant_id, quantity}`
- `PATCH /cart/items/{lineId}` `{quantity}`
- `DELETE /cart/items/{lineId}`

**Checkout & Orders (customer)**
- `POST /checkout` `{fulfilment: "pickup"|"delivery", address?, phone?, points_to_redeem?, idempotency_key}` → `{order_id, paystack_authorization_url}`
- `GET /me/orders?page=`
- `GET /orders/{id}` (owner or staff)
- `GET /orders/{id}/events` (SSE; owner or staff)

**Payments**
- `POST /webhooks/paystack` (signature-verified; no auth middleware — Paystack can't log in)

**Staff/Admin**
- `GET /admin/orders?status=`
- `POST /admin/orders/{id}/transition` `{to: "preparing"}`
- `POST /admin/items` / `PATCH /admin/items/{id}` / `DELETE /admin/items/{id}` (and same for categories, variants)
- `PATCH /admin/items/{id}/availability` `{is_available}`
- `GET /admin/reports/summary` (Phase 2)

**Loyalty (Phase 2)**
- `GET /me/loyalty` → `{balance, ledger[]}`

This surface is the contract with the frontend team. Freeze it early, version changes deliberately, and give them a Postman/Bruno collection or an OpenAPI file in week one so they can build against mocks before the backend is real.

---

## 8. Data Model

```
users            (id, name, email UNIQUE, phone, password_hash, role, token_version, created_at)
categories       (id, name, sort_order)
items            (id, category_id FK, name, description, image_url, is_available, created_at)
item_variants    (id, item_id FK, name, price_pesewas INT, sort_order)
carts            (id, user_id UNIQUE FK)
cart_lines       (id, cart_id FK, item_variant_id FK, quantity, UNIQUE(cart_id, item_variant_id))
orders           (id, user_id FK, status, fulfilment, address, phone,
                  subtotal_pesewas, delivery_fee_pesewas, discount_pesewas, total_pesewas,
                  paystack_reference UNIQUE, idempotency_key UNIQUE, created_at)
order_lines      (id, order_id FK, item_name, variant_name, unit_price_pesewas, quantity)
order_events     (id, order_id FK, from_status, to_status, actor_id NULL FK, created_at)
loyalty_ledger   (id, user_id FK, order_id FK NULL, delta INT, reason, created_at)   -- Phase 2
```

Notes worth defending in a viva:

1. `order_lines` stores **names and prices as copied text/values**, not foreign keys to live menu rows. Menu items get renamed and repriced; historical orders must not retroactively change. (FK to the variant can be kept as a nullable reference for analytics, but display data is snapshotted.)
2. All monetary columns are integer pesewas. The only place currency formatting happens is the frontend.
3. `status` is a CHECK-constrained text column or Postgres enum; the transition rules live in Go (one map), the legal values live in the DB.
4. `paystack_reference` is UNIQUE — this is what makes the webhook idempotent at the database level even if your handler logic has a bug.

---

## 9. Non-Functional Requirements

**Security.** bcrypt for passwords; JWT with short expiry; webhook HMAC verification; parameterized queries only (pgx/sqlc give you this); role middleware on all `/admin` routes; CORS locked to the frontend origin; rate limiting on `/auth/*` (simple in-memory token bucket is fine). No secrets in the repo — given your security positioning, the codebase being clean on a casual audit is part of the deliverable's value to *you* beyond the grade.

**Reliability.** The checkout write (order + lines + ledger redemption) is one DB transaction. The webhook handler is idempotent. The SSE broker must not leak goroutines on client disconnect (respect `r.Context().Done()`).

**Performance.** Irrelevant at this scale and should be stated as such — one shop's traffic is rounding error. The only performance requirement: catalog and order-queue endpoints respond < 300ms locally, which they will unless something is broken.

**Testing.** Unit tests on the two pieces of real logic: the order state machine and checkout total calculation (including points redemption). One integration test for the webhook flow with a faked Paystack signature. Don't aim for coverage numbers; aim for "the money paths are tested."

**Observability.** Structured logging (`log/slog`), request IDs, and log every order state transition. That's enough.

---

## 10. Key Decisions and Tradeoffs (summary)

| Decision | Chosen | Rejected | Why |
|---|---|---|---|
| Real-time | SSE | WebSockets, polling-only | Server→client only; HTTP-native; least code |
| Payment truth | Webhook + verify call | Client callback | Client signals are forgeable |
| Money | Integer pesewas | float64, decimal strings | Floats lose money; ints can't |
| Loyalty | Append-only ledger | Mutable balance column | Auditability, concurrency safety |
| Delivery | Flat fee, free-text address | Zones/geocoding | Zero grading value, high time cost |
| Cart | Server-side | localStorage cart | Survives devices; backend owns pricing |
| DB access | pgx + sqlc | GORM | Explicit SQL; better learning; fewer surprises |
| Architecture | Single binary | Microservices | One shop. Microservices here would be a red flag, not a flex |

---

## 11. Milestones (assumes ~10 working weeks)

| Week | Backend deliverable |
|---|---|
| 1 | Repo, migrations, config, auth (F1). **API contract (OpenAPI/Postman) handed to frontend.** |
| 2–3 | Catalog + cart (F2, F3) deployed to a shared dev environment |
| 4–5 | Checkout + Paystack init + webhook (F4, F5) — the hard part; budget accordingly |
| 6 | Order lifecycle + staff endpoints + SSE (F6, F7) |
| 7 | Admin menu CRUD (F8), order history (F9), hardening, seed data |
| 8 | **Feature freeze on Phase 1.** Integration testing with frontend, demo script rehearsal |
| 9 | Phase 2 (loyalty) only if week 8 exit criteria met; else buffer |
| 10 | Buffer. Something will have eaten a week. This is it. |

Exit criteria for week 8: the full demo script below runs clean twice in a row on the deployed environment, including the webhook path (use Paystack test mode against a tunneled URL — e.g. cloudflared — or the deployed backend).

## 12. Demo Script (the definition of done)

1. Register a new customer; log in.
2. Browse menu; add 2 items (different variants) to cart.
3. Checkout with delivery; pay with Paystack test card.
4. Show order flip to `paid` **without refreshing** (SSE, driven by the webhook).
5. Log in as staff in second window; move order `paid → preparing → ready → out_for_delivery → completed`; customer screen updates live at each step.
6. Show customer order history.
7. Admin: toggle an item unavailable; show it can't be added to cart.
8. (Phase 2) Show points earned; place second order redeeming points.

## 13. Risks

| Risk | Mitigation |
|---|---|
| Webhook can't reach localhost during dev | cloudflared/ngrok tunnel from day one of payment work; deploy early |
| Frontend blocked waiting on backend | API contract + mock responses in week 1 |
| Scope creep from teammates/lecturer | This document. Point at §2 Non-Goals and §4 phasing |
| Walker's parallel projects eating weeks 4–5 | Paystack integration is scheduled in two weeks for a reason; it cannot be a weekend job |
| Paystack GHS test quirks | Read their test-mode docs before week 4, not during |

## 14. Open Questions for the Team

1. Where does this deploy? (Backend: any box that runs a binary + Postgres — Railway/Render/Fly free tiers all work. Decide week 1, deploy week 2.)
2. Does the rubric reward documentation/testing explicitly? If yes, §9 testing scope may expand.
3. Who owns seed data (realistic menu with images)? Suggest frontend, since they need it for design anyway.
4. Is guest checkout actually wanted by the lecturer, or assumed? Ask before building auth walls into every flow.

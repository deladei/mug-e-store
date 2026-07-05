# Architecture

A single Go binary (stdlib `net/http`, no framework) over PostgreSQL (`lib/pq`,
no ORM), with an in-process SSE broker for real-time order status. One service,
one database, single-tenant. Money is integer pesewas everywhere.

Decisions and deviations are recorded as ADR rows in `DECISIONS.md`; the
production-readiness triage is in `docs/production-readiness.md`.

## Layering (strict — a layer calls only the one below it)

```
            HTTP / JSON
                │
   cmd/api  ───────────────  composition & process lifecycle only
                │
 internal/httpapi  ────────  routing, middleware, HTTP↔function translation
                │
 internal/store   ────────  the ONLY place SQL is written
                │
 internal/domain  ────────  ALL business rules (order FSM, redemption math)
                │
            PostgreSQL
```

Supporting packages used by `httpapi`: `internal/auth` (bcrypt + JWT + token
hashing), `internal/paystack` (Initialize/Verify + HMAC signature verify),
`internal/sse` (per-order pub/sub broker), `internal/config` (env loader).

The rule that pays off: **business rules never leak upward**. A handler cannot
decide whether an order may move to `paid`; it asks `domain`. The store cannot
invent a price; it snapshots what `domain`/checkout computed.

## Request middleware chain

```
recoverPanic → securityHeaders → logRequests → cors → mux → handler
                                                         ├─ requireAuth / requireRole / requireAuthQuery
                                                         └─ rateLimit (credential endpoints)
```

## Order lifecycle (fulfilment-aware FSM, owned by internal/domain)

```
 pending_payment ──paid──▶ paid ──▶ preparing ──▶ ready ──▶ completed   (pickup)
        │                                   │
        │                                   └──▶ out_for_delivery ──▶ completed   (delivery)
        └───────────────── cancelled ◀── (any pre-completion state)
```

- Pickup orders never enter `out_for_delivery`; delivery orders never reach
  `completed` without shipping first. Illegal jumps are rejected.
- Transitioning to the status already held is a successful **no-op** — this is
  what makes webhook retries safe.
- Loyalty is earned only on the `completed` transition; cancelling a redeemed
  order writes a compensating refund ledger row.

## Payment truth (the only path to `paid`)

```
 Paystack ──webhook──▶ POST /api/v1/webhooks/paystack
                          │
              ┌───────────┴─── gate 1: valid HMAC-SHA512 signature
              ├─────────────── gate 2: server-to-server Verify returns success
              ├─────────────── gate 3: verified amount == stored total AND currency == GHS
              └─────────────── gate 4: the transition is legal (domain FSM)
                          │
                 all pass ▼
                    order → paid  (then SSE notifies the customer)
```

No client and no staff member can set an order to `paid` by hand. The webhook is
idempotent (`paystack_reference` is `UNIQUE`; `paid→paid` is a no-op): a `200`
tells Paystack to stop, a `5xx` tells it to retry — both are safe.

## Real-time status (SSE)

`GET /api/v1/orders/{id}/events` subscribes the customer to an in-process broker
keyed by order id (token passed as a query param because `EventSource` cannot set
headers). Each state change publishes to subscribers. Non-blocking publish,
leak-free unsubscribe, no Redis or external queue — appropriate for one instance;
a multi-replica deploy would move this to a shared pub/sub (noted as a scaling
step, not a current requirement).

## Persistence model (11 tables, four groups)

- **identity:** `users`, `refresh_tokens` (+ `password_reset_tokens`)
- **catalog:** `categories`, `items`, `item_variants` (price lives on the variant)
- **cart:** `carts`, `cart_lines` (stores no price — resolved live)
- **orders + ledger:** `orders`, `order_lines` (immutable snapshot), `order_events`
  (append-only audit), `loyalty_ledger` (append-only; balance = `SUM(delta)`)

See `docs/coffee-mug-shop-database-schema.md` for column-level detail.

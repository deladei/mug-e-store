# Coffee Mug Shop — User Journey Flow (Backend-Centered)

**What this document is:** the journeys a user takes through the app, with each step mapped to the **endpoint behind it**, what the backend does, and what it returns. It is the narrative of the API in use. For the visual flow, see the FigJam board; for screen-by-screen response shapes, see the API Consumption Brief.

**Three actors:** Customer, Staff (barista), Admin (owner). The Paystack hosted page and the Paystack webhook are external participants that drive two of the most important transitions.

**Version:** 1.0

---

## 1. Customer journey: browse → order → pay → track

This is the critical path — the loop the whole system exists to close.

### Step 1 — Browse the menu
- **Calls:** `GET /api/v1/categories`, then `GET /api/v1/items` (optionally `?category={id}`).
- **Backend does:** returns categories ordered by `sort_order`; returns items with their variants nested, **filtering out unavailable items** for anonymous/customer callers. (A staff token would include unavailable items; a customer never sees them.)
- **Returns:** arrays of catalog data with `price_pesewas` per variant.
- **Note:** browsing requires no account. The auth wall comes later, at checkout.

### Step 2 — Inspect one item
- **Calls:** `GET /api/v1/items/{id}`.
- **Backend does:** returns the single item with all variants.
- **Returns:** `404 not_found` if the item does not exist or is unavailable to this caller.

### Step 3 — Build the cart
- **Calls:** `POST /api/v1/cart/items` with `{item_variant_id, quantity}`; adjust with `PATCH /api/v1/cart/items/{lineId}`; remove with `DELETE`.
- **Auth required from here on.** If the customer is not logged in, the frontend must route them through Step 4 first.
- **Backend does:** creates the user's cart on first add (idempotently). Adding the same variant twice **increments quantity** rather than creating a duplicate line. Rejects an unavailable variant with `409 unavailable`. The cart lives on the server, keyed to the user — not in the browser — so it survives across devices and refreshes.
- **Returns:** the full cart with a server-computed `subtotal_pesewas`. Prices are resolved **at read time from the live menu**, so a price change is reflected immediately and consistently.

### Step 4 — Authenticate (gate, can happen anywhere before checkout)
- **Calls:** `POST /api/v1/auth/register` or `POST /api/v1/auth/login`.
- **Backend does:** on register, validates email and an 8+ character password, hashes with bcrypt, creates the user. On login, verifies the password; returns an identical error for unknown-email and wrong-password (no enumeration). On success, issues a 15-minute access token (in the JSON body) and a 7-day rotating refresh token (as an `httpOnly` cookie).
- **Returns:** `{access_token, user}`. The frontend holds the access token in memory and sends it as `Authorization: Bearer …` on every authenticated call.

### Step 5 — Checkout
- **Calls:** `POST /api/v1/checkout` with `{fulfilment, address?, phone?, idempotency_key?}`.
- **Backend does, in one transaction:** re-reads the cart at current prices; rejects an empty cart (`400 empty_cart`) or a now-unavailable item (`409 unavailable`); computes subtotal; adds the flat delivery fee **only if** `fulfilment = delivery` (which also requires address and phone); writes an order in status `pending_payment`; snapshots every line into immutable `order_lines`; clears the cart. Then it calls Paystack `initialize` with the order total and a generated `CMUG-…` reference. If Paystack initialization fails, the order is immediately **cancelled** so it cannot dangle, and the customer gets `502 payment_init_failed`.
- **Returns:** `{order, authorization_url}`. The `authorization_url` is Paystack's hosted payment page.

### Step 6 — Pay (external: Paystack hosted page)
- The frontend redirects the customer to `authorization_url`. **All card entry happens on Paystack's domain.** The backend never sees card data — this is a security property, not a limitation.
- On completion Paystack redirects the customer back to the frontend callback (`/orders/{id}`).

### Step 7 — Payment confirmation (external: webhook — the real source of truth)
- **Paystack calls:** `POST /api/v1/webhooks/paystack`.
- **Backend does:** verifies the HMAC-SHA512 signature → calls Paystack `verify` server-to-server → checks the verified amount equals the order total and currency is GHS → transitions the order `pending_payment → paid` → publishes the new status to the SSE broker. Retried webhooks are safe (idempotent no-op). A mismatch is logged loudly and the order is **not** marked paid.
- **Returns (to Paystack):** `200 ok`. On a transient verify failure it returns `500` so Paystack retries.
- **Important:** the customer's browser returning to the callback page does **not** mark the order paid. Only this webhook path does. The callback page simply opens the tracking view and waits.

### Step 8 — Track the order live
- **Calls:** `GET /api/v1/orders/{id}/events` as an `EventSource`, passing the token in the query string: `new EventSource('/api/v1/orders/{id}/events?token=' + accessToken)`. (EventSource cannot set an Authorization header, so the backend accepts `?token=` for this one endpoint.)
- **Backend does:** authorizes ownership (a customer can only stream their own order; foreign IDs return `404`), immediately sends a **snapshot** of the current status so a late-connecting client is never blank, then pushes a `status` event on every transition, with a heartbeat comment every 25 seconds to keep proxies from closing the stream. On disconnect, the subscription is cleaned up (no goroutine leak).
- **Returns:** a stream of `event: status\ndata: {"order_id":…,"status":"preparing"}`.
- **Fallback:** the same data is available by polling `GET /api/v1/orders/{id}`, so the UI degrades gracefully if SSE is unavailable.

### Step 9 — After completion
- When staff complete the order, the customer's stream receives a final `completed` status, and (Phase 2) loyalty points appear via `GET /api/v1/me/loyalty`. Order history is available at `GET /api/v1/me/orders`.

---

## 2. Staff journey: work the order queue

### Step 1 — Sign in
- **Calls:** `POST /api/v1/auth/login`. The returned token carries `role: staff`.

### Step 2 — See the queue
- **Calls:** `GET /api/v1/admin/orders` (optionally `?status=paid`).
- **Backend does:** with no filter, returns all **active** orders (everything except `completed`, `cancelled`, and `pending_payment` — i.e. paid-and-onward, the orders that need action), oldest first. With a status filter, returns exactly that status.
- **Returns:** an array of orders with totals, fulfilment type, and delivery address/phone where relevant.

### Step 3 — Advance an order
- **Calls:** `POST /api/v1/admin/orders/{id}/transition` with `{to: "preparing"}` (then `ready`, then `completed` or `out_for_delivery`).
- **Backend does:** takes a row lock, checks the move is legal **for that order's fulfilment type** via the domain state machine, writes the new status, records an audit event with the staff member's ID, and — on `completed` — appends loyalty points to the customer's ledger. Then publishes the new status to any open customer SSE stream. An illegal move returns `409 invalid_transition`. Attempting to set `paid` by hand returns `403` — only the payment webhook can do that.
- **Returns:** the updated order.

### Step 4 — Inspect order history
- **Calls:** `GET /api/v1/admin/orders/{id}/history`.
- **Backend does:** returns the ordered audit trail of every status change (`from_status → to_status`, who, when) from `order_events`.

### Step 5 — Toggle item availability (sold out)
- **Calls:** `PATCH /api/v1/admin/items/{id}/availability` with `{is_available: false}`.
- **Backend does:** flips the flag; the item immediately disappears from customer browsing and can no longer be added to carts.

---

## 3. Admin journey: manage the menu

Admin has every staff capability plus catalog management.

- **Categories:** `POST /admin/categories`, `PATCH /admin/categories/{id}`, `DELETE /admin/categories/{id}`.
- **Items:** `POST /admin/items`, `PATCH /admin/items/{id}`, `DELETE /admin/items/{id}`.
- **Variants:** `POST /admin/items/{id}/variants`, `DELETE /admin/variants/{id}`.
- **Backend does:** standard create/update/delete with validation (non-empty names, non-negative prices). Deleting an item cascades to its variants and cart lines; **past orders are untouched** because their lines were snapshotted.

---

## 4. The two transitions that are not user-driven

Most steps above are triggered by a person tapping something. Two are triggered by the system, and they are the ones to understand deeply for the viva:

1. **`pending_payment → paid`** is triggered **only** by the verified Paystack webhook (§1, Step 7). No human and no client can cause it.
2. **Loyalty earn** is triggered as a side effect of the **`→ completed`** transition (§2, Step 3), inside the same locked transaction, so points and order completion can never disagree.

---

## 5. Failure and edge paths (what the backend does when things go wrong)

| Situation | Backend response |
|---|---|
| Checkout with empty cart | `400 empty_cart` |
| Item went unavailable between add and checkout | `409 unavailable`, order not created |
| Paystack initialize fails | order auto-cancelled, `502 payment_init_failed` |
| Customer abandons payment | order stays `pending_payment`; no points, no queue entry |
| Webhook signature invalid | `401`, order untouched, logged as a warning |
| Webhook amount ≠ order total | order **not** paid, logged as an error for manual review |
| Duplicate webhook delivery | idempotent no-op, `200` |
| Duplicate checkout (same idempotency key) | `409 duplicate_order` |
| Customer requests another user's order | `404` (never `403`) |
| Staff tries an illegal status jump | `409 invalid_transition` |
| Staff tries to set `paid` manually | `403 forbidden` |

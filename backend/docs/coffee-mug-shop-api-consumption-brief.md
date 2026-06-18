# Coffee Mug Shop — API Consumption Brief

**What this document is, and why it is named this way:** you asked for a "UI/UX brief," but the backend is the only side you own, and a visual wireframe brief already exists. So this is the version that earns its place from the backend: for every screen the frontend team will build, it states **which endpoints power it, what JSON comes back, which authentication to use, and which states the backend can actually produce** (loading, empty, every error). It is the contract your teammates code against. Hand it to them with the wireframe brief — that one says what the screens look like, this one says what data fills them.

**Conventions used below**
- Base path `/api/v1`. All request/response bodies are JSON.
- **Money** is always integer **pesewas**. The frontend divides by 100 and formats (e.g. `3000` → "GHS 30.00"). Never do money math in floats.
- **Auth methods:** `Bearer` = send `Authorization: Bearer {access_token}`. `Cookie` = the `httpOnly` refresh cookie travels automatically (the frontend cannot read it). `Query token` = SSE only, `?token={access_token}`.
- **Error envelope (every error, everywhere):** `{"error": {"code": "...", "message": "..."}}`. Branch UI logic on `code`, show `message` to the user only as a fallback.

**Version:** 1.0

---

## 1. Cross-cutting: the session model the frontend must implement

This is the one piece of shared plumbing every authenticated screen depends on.

- On login/register the backend returns `{access_token, user}` **and** sets the refresh cookie. Hold `access_token` **in memory** (React state/context) — not in `localStorage` (XSS exposure).
- The access token expires in **15 minutes**. When any call returns `401 unauthorized`, the frontend should call `POST /auth/refresh` (cookie travels automatically), get a fresh `access_token`, and retry the original call once. If the refresh itself returns `401`, the session is over — route to login.
- `POST /auth/logout` clears the cookie server-side; the frontend also drops the in-memory token.

Build this as an interceptor once. Every screen below assumes it exists.

---

## 2. Customer screens

### 2.1 Menu / storefront
- **Powered by:** `GET /categories`, `GET /items` (`?category={id}` to filter). **Auth:** none.
- **Receives:** categories as `[{id, name, sort_order}]`; items as `[{id, category_id, name, description, image_url, is_available, variants: [{id, name, price_pesewas, sort_order}]}]`.
- **States to build:**
  - *Loading* — while fetching.
  - *Empty category* — a category with no items returns `[]`, not an error.
  - *Item display price* — show the lowest variant price as "from GHS X" if an item has multiple variants.
- **Note:** customers never receive unavailable items, so there is no "sold out" state to render on this screen — the item is simply absent.

### 2.2 Item detail
- **Powered by:** `GET /items/{id}`. **Auth:** none.
- **Receives:** one item object with its `variants` array.
- **States:** *not found* → `404 not_found` (item missing or unavailable) → show a "no longer available" message and a link back. The **variant selector is required** before "Add to cart" can be enabled — the cart references `item_variant_id`, never an item.

### 2.3 Cart
- **Powered by:** `GET /cart`; mutate with `POST /cart/items {item_variant_id, quantity}`, `PATCH /cart/items/{lineId} {quantity}`, `DELETE /cart/items/{lineId}`. **Auth:** Bearer.
- **Receives:** `{lines: [{line_id, item_variant_id, item_name, variant_name, unit_price_pesewas, quantity, available}], subtotal_pesewas}`.
- **States to build:**
  - *Empty cart* — `lines: []`, `subtotal_pesewas: 0`. Show an empty state, disable checkout.
  - *Unavailable line* — a line may come back with `available: false` if the item was turned off after it was added. Flag it visually and block checkout until removed; the backend will reject checkout otherwise.
  - *Add returns `409 unavailable`* — the item went off-menu; show a toast.
  - Every mutation returns the **full updated cart**, so the frontend can replace its state from the response rather than re-fetching.
- **Auth gate:** all cart calls need a token. A not-logged-in user tapping "Add" should be routed to auth first (§2.4), then returned.

### 2.4 Auth (login / register / guest)
- **Powered by:** `POST /auth/register {name, email, phone, password}` or `POST /auth/login {email, password}`. **Auth:** none (this is where it begins).
- **Guest checkout:** `POST /auth/guest {name?, phone?}` mints a passwordless guest session so a shopper can order without creating an account. The body is **optional** — both fields may be omitted (a fully anonymous guest shows as "Guest"); the form should still collect `name`/`phone` so staff have someone to call out and a number to reach. **Auth:** none (rate limited). It returns the **same** `{access_token, user, refresh cookie}` shape as login, so every downstream call (cart, `/checkout`, order tracking, history) is identical to a logged-in customer — no special-casing in the client. Caveats: a guest **cannot log in later** (it has no password) and **earns no loyalty points**, so don't show a "log in to your guest account" or points affordance in the guest flow. The checkout contract is unchanged — the order's own `address`/`phone` still come from `POST /checkout`.
- **Receives:** `{access_token, user: {id, name, email, phone, role, created_at}}` + refresh cookie set.
- **States to build:**
  - *Validation* — register returns `400 validation` if password < 8 chars or email malformed; surface inline.
  - *Email taken* — `409 email_taken` on register.
  - *Bad credentials* — `401 invalid_credentials` on login. The message is intentionally identical for unknown-email and wrong-password; do not try to distinguish them.
  - *Rate limited* — `429 rate_limited` after 10 attempts/minute from one IP; show "too many attempts, wait a moment."

### 2.5 Checkout
- **Powered by:** `POST /checkout {fulfilment, address?, phone?, idempotency_key?, points_to_redeem?}`. **Auth:** Bearer.
- **Send:** `fulfilment` is `"pickup"` or `"delivery"`. For delivery, `address` and `phone` are **required** (backend rejects otherwise). Generate a UUID `idempotency_key` per checkout attempt and reuse it on retry of the *same* attempt. `points_to_redeem` is optional (omit or `0` for none) — see loyalty below.
- **Receives:** `201 {order, authorization_url}`. The returned `order` carries `discount_pesewas` (points applied) and the discounted `total_pesewas`; that total is exactly what Paystack will charge.
- **Then:** redirect the browser to `authorization_url` (Paystack's hosted page). Do **not** build a card form — card entry is entirely on Paystack's domain.
- **States to build:**
  - *Delivery fee* — when the user toggles to delivery, show the flat fee (the order's `delivery_fee_pesewas`) added to the total. The fee is a backend config value; read it from the returned order, do not hardcode it.
  - *Points applied* — if `points_to_redeem` was sent, show `discount_pesewas` as a line and the reduced `total_pesewas`.
  - *Empty cart* — `400 empty_cart`.
  - *Item unavailable* — `409 unavailable`; send the user back to the cart.
  - *Not enough points* — `409 insufficient_points` if `points_to_redeem` exceeds the customer's balance.
  - *Duplicate* — `409 duplicate_order` if the same idempotency key was already used.
  - *Payment could not start* — `502 payment_init_failed`; the order was auto-cancelled, tell the user to try again.

### 2.6 Payment return / live order tracking
- **Powered by:** `GET /orders/{id}/events` via `EventSource`. **Auth:** **Query token** — `new EventSource('/api/v1/orders/' + id + '/events?token=' + accessToken)`. (EventSource cannot send headers; this endpoint is the one place a query-string token is accepted.)
- **Receives:** a stream of `event: status` messages whose `data` is `{order_id, status}`. The **first message is always a snapshot** of current status, so the stepper can render immediately even if payment already completed before the page opened.
- **States to build, mapped to `status`:** a stepper over `pending_payment → paid → preparing → ready → (out_for_delivery →) completed`. Show `pending_payment` as "confirming payment." Show `cancelled` as a terminal failed state.
- **Token expiry on a long stream:** the access token can expire while the stream is open. On the stream erroring, refresh the token and reconnect a new `EventSource`.
- **Fallback (build this too):** if `EventSource` fails or is unsupported, poll `GET /orders/{id}` every few seconds for the same `{…, status}`. The data shape is a superset, so the stepper logic is shared.

### 2.7 Order history
- **Powered by:** `GET /me/orders?page={n}`. **Auth:** Bearer.
- **Receives:** `{orders: [...], page}` — 20 per page, newest first, each with its snapshotted `lines`.
- **States:** *empty* (`orders: []`), *pagination* (request next page when 20 returned).

### 2.8 Profile / loyalty
- **Powered by:** `GET /me/loyalty`. **Auth:** Bearer.
- **Receives:** `{balance, ledger: [{order_id, delta, reason, created_at}]}`. `balance` is total points; positive `delta` is earned (`earn_on_completion`) or refunded (`refund_on_cancel`), negative is redeemed (`redeem_at_checkout`).
- **States:** *zero balance* for new customers.
- **Redeeming at checkout (now live):** points are worth **1 pesewa each** (100 points = GHS 1). Send `points_to_redeem` on `POST /checkout` (§2.5). The discount is **capped at the subtotal** — points reduce the cost of the coffee, never the delivery fee — so the most a customer can usefully redeem is `min(balance, subtotal_pesewas)`; compute that to bound the control and show the resulting `discount_pesewas`/`total_pesewas` from the returned order. Over-redeeming the balance returns `409 insufficient_points`. If a redeemed order is later cancelled, the points are automatically refunded (a `refund_on_cancel` ledger entry), so the balance self-heals.

### 2.9 Password reset (now live)
- **Powered by:** `POST /auth/password-reset/request {email}` then `POST /auth/password-reset/confirm {token, password}`. **Auth:** none (you're locked out — that's the point). Both are rate-limited (`429 rate_limited`, 10/min per IP).
- **Request step:** always returns `200 {message}` — **the same response whether or not the email exists** (no account enumeration). Build a single "if that address has an account, we've sent a link" confirmation screen; never branch the UI on whether the email was found.
- **The link:** the user receives a link to your reset page carrying the token, e.g. `{FRONTEND_ORIGIN}/reset-password?token=…`. Build a `/reset-password` route that reads `token` from the query string and posts it with the new password to the confirm step. *(Backend note: email delivery is wired at deploy time; until then the link is in the server logs — see STATE.md.)*
- **Confirm step:** send `{token, password}` (the new password, **min 8 chars** — `400 validation` otherwise). On success: `200 {message}`. The token is **single-use and expires after 1 hour**; an unknown, already-used, or expired token all return `400 invalid_token` — show "this reset link is invalid or has expired, request a new one" and route back to the request step.
- **After a successful reset, every existing session is revoked** (all refresh tokens are dropped server-side). The user must log in again with the new password — there is no auto-login from this flow.

---

## 3. Staff screens

### 3.1 Order queue
- **Powered by:** `GET /admin/orders` (all active) or `?status={status}` (one column). **Auth:** Bearer (role staff/admin).
- **Receives:** `{orders: [...]}` oldest-first, each with `status`, `fulfilment`, totals, and delivery `address`/`phone`.
- **States:** *empty queue*; group cards by `status` for a board layout (request each status, or fetch all-active and group client-side).

### 3.2 Order detail + advance
- **Powered by:** `GET /orders/{id}` (full order incl. lines), `GET /admin/orders/{id}/history` (the event timeline), `POST /admin/orders/{id}/transition {to}`. **Auth:** Bearer (staff/admin).
- **Advance buttons:** the legal "next" buttons depend on **fulfilment type**:
  - pickup: `paid → preparing → ready → completed`
  - delivery: `paid → preparing → ready → out_for_delivery → completed`
- **States to build:**
  - *Illegal move* — `409 invalid_transition` (e.g. trying to skip a step). Disable buttons that aren't the legal next step rather than relying on the error.
  - *No "mark paid" button* — `paid` is reachable only via the payment webhook; the backend returns `403` if staff attempt it. Do not render that control.
  - *History timeline* — render `from_status → to_status`, actor, timestamp. A `null` actor means the system (the payment webhook) made the change.

### 3.3 Sold-out toggle
- **Powered by:** `PATCH /admin/items/{id}/availability {is_available}`. **Auth:** Bearer (staff/admin).
- **Effect:** immediately hides/shows the item for customers. Reflect the new state from the response.

---

## 4. Admin screens (menu management)

- **Categories:** `POST /admin/categories {name, sort_order}`, `PATCH /admin/categories/{id}`, `DELETE`. **Auth:** Bearer (admin only).
- **Items:** `POST /admin/items {category_id, name, description, image_url}`, `PATCH /admin/items/{id}`, `DELETE`.
- **Variants:** `POST /admin/items/{id}/variants {name, price_pesewas, sort_order}`, `DELETE /admin/variants/{id}`.
- **States to build:** *validation* (`400` for empty name / negative price / missing category), *duplicate category* (`409 duplicate`), *not found* (`404`). The item editor must support **repeatable variant rows**, since price lives on the variant, and a price input collects **pesewas** (or collect cedis in the UI and multiply by 100 before sending).
- **Non-admin staff** calling these get `403 forbidden`; hide the menu-management surface for the staff role.

### 4.1 Admin reports (dashboard)
- **Powered by:** `GET /admin/reports/summary?days={n}`. **Auth:** Bearer (**admin only** — staff get `403`). `days` is optional (default 30, clamped to 1–365).
- **Receives:**
  ```json
  {
    "from": "2026-05-15", "to": "2026-06-13",
    "totals": { "orders": 42, "paid_orders": 38, "revenue_pesewas": 95000 },
    "daily": [ { "date": "2026-05-15", "orders": 0, "paid_orders": 0, "revenue_pesewas": 0 }, … ]
  }
  ```
- **Notes for the frontend:** `daily` is **continuous** — every day in the window is present, gap days zero-filled — so plot it straight onto a chart with no gap handling. `orders` counts all orders placed that day; `paid_orders`/`revenue_pesewas` count only **confirmed** orders (payment taken, not cancelled) — pending and cancelled orders never contribute to revenue. Money is pesewas; format for display.

---

## 5. Quick reference: auth method per endpoint group

| Endpoint group | Auth method |
|---|---|
| `/categories`, `/items*` (read) | none |
| `/auth/register`, `/auth/login`, `/auth/guest` | none (rate limited) |
| `/auth/refresh`, `/auth/logout` | Cookie |
| `/cart*`, `/checkout`, `/me/*`, `GET /orders/{id}` | Bearer |
| `GET /orders/{id}/events` (SSE) | Query token (`?token=`) |
| `/webhooks/paystack` | none from frontend — Paystack only |
| `/admin/*` (queue, transition, availability) | Bearer, role staff or admin |
| `/admin/categories|items|variants` (write) | Bearer, role admin |
| `GET /admin/reports/summary` | Bearer, role admin (financial data) |

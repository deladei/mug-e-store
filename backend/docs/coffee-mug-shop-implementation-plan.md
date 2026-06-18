# Coffee Mug Shop — Backend Implementation Plan

**An honest framing first.** The backend is already built, compiling, and passing its tests. So this is **not** a plan to build from zero — that would be re-narrating finished work, which is the exact "planning instead of executing" trap worth avoiding. This is an **execution plan for what genuinely remains**: the one feature deliberately left for you, the tests this sandbox cannot run, deployment, and the sequencing of integration with the frontend team. The gaps are named plainly so the plan describes reality.

**Version:** 1.0

---

## 1. Where things actually stand

### Done and verified
- Full layered Go service (domain → store → httpapi), single binary.
- Identity: register/login, bcrypt, JWT access tokens, rotating hashed refresh tokens.
- Catalog read + full admin CRUD (categories, items, variants), availability toggle.
- Server-side cart with quantity-merging and availability enforcement.
- Checkout as a single transaction: snapshot lines, clear cart, initialize Paystack, auto-cancel on init failure.
- Payment webhook with the four-gate verification (signature → verify → amount/currency → legal transition), idempotent against retries.
- Order state machine, fulfilment-aware, with an audit trail.
- Loyalty **earning** on completion (append-only ledger).
- Real-time order status over SSE (snapshot-first, heartbeat, leak-free cleanup).
- Migrations, seed data, admin/staff seeding helper, Makefile, `.env.example`.
- Tests passing: exhaustive state-machine unit tests; webhook signature tests (forged / valid / wrong-secret / tampered).

### Built but only proven against a mock
- The Paystack initialize + verify round trip was exercised against a local mock server, not the real Paystack API.

### Deliberately not built (yours to own)
- **Loyalty redemption at checkout** — earning works; spending does not. This is your task (§3).

### Cannot be done from the build sandbox (yours to run on real infrastructure)
- A true end-to-end Paystack test with real test keys and a publicly reachable webhook URL (§4).
- Deployment (§6).

---

## 2. Immediate next step: get it running on your machine

Before any new code, confirm the delivered project runs in your own environment. This is the first thing to do in Claude Code on your laptop.

1. Unzip the project into your repo.
2. Ensure PostgreSQL is running locally; create a database and user.
3. Copy `.env.example` to `.env` and fill in `DATABASE_URL`, a long random `JWT_SECRET`, and your Paystack **test** secret key.
4. `make migrate-up` then `make seed` (creates the menu plus `admin@coffeemug.shop` / `staff@coffeemug.shop`, password `password123`).
5. `make run`, then `curl localhost:8080/api/v1/healthz` — expect `{"status":"ok"}`.
6. Read `internal/domain/order.go` end to end. It is 67 lines and it is the heart of the system; you will be asked about it in the viva.

**Definition of done for this step:** the demo loop in the PRD runs locally against your own database.

---

## 3. Task you own: loyalty redemption at checkout

This is specified enough to implement directly, and it is the right piece to write yourself because it touches money, transactions, and the ledger — the parts you most need to be able to defend.

**Goal:** let a customer spend points to reduce an order total at checkout. The earn side and the schema already support it (`loyalty_ledger.delta` is signed; `orders.discount_pesewas` exists and is currently always 0).

**Where it goes:** extend `store.Checkout` and the checkout handler. Do **not** put the rule anywhere except the transaction.

**The rules to implement and defend:**
1. Accept an optional `redeem_points` in the checkout request.
2. Inside the **same transaction** as the order insert, compute the customer's current balance (`SUM(delta)`). Reject if they are trying to redeem more than they have.
3. Convert points to a discount using a fixed rule (PRD: 100 points = GHS 1 = 100 pesewas). Cap the discount so the total can never go below zero (decide and document: can points cover the delivery fee, or only the subtotal?).
4. Set `discount_pesewas` on the order; `total_pesewas = subtotal + delivery_fee − discount`.
5. Write a **negative** `delta` ledger row (`reason: 'redeem_at_checkout'`) in the same transaction, so the spend is atomic with the order.
6. The amount sent to Paystack is the **discounted** total, and the webhook amount check still compares against the stored `total_pesewas` — so this must remain internally consistent.

**The trap to avoid:** double-spend. If you compute the balance, then insert the order, then write the ledger row in three separate statements without the transaction and lock, two concurrent checkouts could each see the same balance and both redeem it. The transaction plus reading the balance inside it is what prevents this — the same pattern `Transition` already uses with `FOR UPDATE`.

**Test it:** redeem exactly the balance; redeem more than the balance (must fail); two concurrent redemptions of the same points (only one may succeed).

---

## 4. Task you own: real end-to-end Paystack test

The mock proves the *logic*; only real keys prove the *integration*.

1. In the Paystack dashboard (test mode), get your test secret key into `.env`.
2. Expose your local server with a tunnel (e.g. `ngrok http 8080`) so Paystack can reach your webhook.
3. Set the webhook URL in the Paystack dashboard to `{tunnel}/api/v1/webhooks/paystack`.
4. Run a real checkout, pay with a Paystack **test card**, and confirm: the webhook arrives, the signature passes, the verify call succeeds, the order flips to `paid`, and your open SSE stream receives the event.
5. Deliberately test the failure you cannot easily unit-test: let the verify call see a pending payment and confirm the order does **not** flip.

**This is the highest-risk integration in the project and the one most likely to eat time** (it is flagged as such in the PRD milestones). Do it early, not in the final week.

---

## 5. Task you own: fill out the test suite

The domain and signature layers are well tested. Add, in priority order:
1. **Store tests against a real test database** — checkout (happy path, empty cart, unavailable item, duplicate idempotency key), and `Transition` (legal, illegal, idempotent no-op, loyalty earn on completion).
2. **HTTP-level tests** with `net/http/httptest` — auth flow, the ownership `404`, the staff `403` on manual `paid`.
3. Redemption tests from §3.

---

## 6. Task you own: deployment

A single Go binary plus a Postgres database. Options, simplest first:
- A platform with managed Postgres (e.g. Render, Railway, Fly.io). Build the binary, set the environment variables, point it at the managed database, run the migration once.
- Set `FRONTEND_ORIGIN` to the deployed frontend's URL, and switch the refresh cookie to `SameSite=None; Secure` if frontend and backend are on different origins (noted in the code and the README).

---

## 7. Sequencing with the frontend team

Your teammates are blocked on contracts, not on your internal code. Unblock them in this order:

1. **Now:** hand over the API Consumption Brief, the User Journey, and the running server (or its deployed URL). That is everything they need to start every screen.
2. **Parallel work:** they build menu, cart, and auth screens against the live read/write endpoints while you implement redemption (§3) and run the real Paystack test (§4) — these do not block each other.
3. **Integration point 1:** checkout + payment redirect. Coordinate the `authorization_url` redirect and the callback route (`/orders/{id}`) with them once §4 works.
4. **Integration point 2:** live tracking. Give them the exact `EventSource` snippet (it is in the Consumption Brief) and confirm the `?token=` auth and reconnect-on-expiry behaviour together.
5. **Freeze:** once the demo loop runs end to end across both sides, stop adding features and harden.

---

## 8. Milestone shape (anchor to your real remaining weeks)

The PRD assumed ~10 working weeks and you never confirmed the actual number — so this is **relative ordering**, not invented dates. Slot it against your true timeline.

| Order | Milestone | Gate (done when…) |
|---|---|---|
| 1 | Runs locally | demo loop works against your DB |
| 2 | Real Paystack round trip | a test-card payment flips the order to paid via the webhook |
| 3 | Redemption built + tested | concurrent double-spend test passes |
| 4 | Frontend integrated | checkout → pay → live tracking works across both sides |
| 5 | Deployed | the same loop works on the deployed URL |
| 6 | Hardened + test suite filled | store + HTTP tests green; freeze |

The single most important thing in this table is **Milestone 2 happening as early as possible.** Payment integration is where this class of project usually dies, and you cannot compress it by planning — only by doing it and hitting the real errors. Everything else here you have already shown you can do.

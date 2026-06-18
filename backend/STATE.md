# STATE

Single source of "where we are" and "what's next". Read this at session start.

## Current status

- **Session:** 3 — ✅ complete (2026-06-14). Next is Session 4. This session: shipped Phase 2 **S3 password reset** (backend-only — request + confirm endpoints, hashed single-use 1h tokens, session-revocation-on-reset, no enumeration), migration `0002`, store + HTTP tests. Working tree clean.
- **Session 2:** ✅ complete (2026-06-13). Finished the httpapi webhook, added `cmd/api`, shipped loyalty redemption, wrote the full DB-backed + HTTP test suites, kept the monorepo PR mirrored.
- **Runnable:** ✅ `cmd/api` binary builds and boots; fails fast with an aggregated config error when required env is unset.
- **Loyalty redemption:** ✅ feature complete + DB-tested. Pure `domain.Redemption` rule (9 unit cases); `store.Checkout` redeems inside the txn under a user-row `FOR UPDATE` lock; `TransitionOrder` refunds on cancel. Contract published to the API brief.
- **Build:** `go build ./...` clean; `go vet ./...` clean.
- **Tests:** `go test ./...` green; **with `TEST_DATABASE_URL` set, `go test -p 1 ./...` green** (incl. DB-backed store + HTTP-level; password-reset store + HTTP suites added Session 3). Without the var the store/httpapi DB suites skip, so default `go test ./...` stays green on a box without Postgres. **NB: Session 3 added migration `0002` — a reset of the test box needs both `0001` and `0002` applied (see Test DB below).**
- **Test DB:** local Postgres 18 on socket; created `coffeemug_test` (owner `walker`) with `0001_init` **and `0002_password_reset_tokens`** applied. DSN used: `host=/var/run/postgresql user=walker dbname=coffeemug_test sslmode=disable` (peer auth — TCP needs a password, the socket does not). To rebuild: `createdb coffeemug_test` then apply both migration up files in order.
- **Phase 0 (scaffolding):** ✅ committed `e83a81c`.
- **Phase 1 step 1 (schema migration):** ✅ committed `129defd`. Verified against Postgres (11 tables up, 0 down, re-up clean).
- **Phase 1 step 2 (domain state machine):** ✅ committed `890e349`. Exhaustive table-driven tests green (`internal/domain`).
- **Phase 1 step 3 (config loader):** ✅ committed `f48982f`. `internal/config` — env-only, refuses to boot without required secrets; tests green.
- **Phase 1 step 4 (auth):** ✅ committed `351c75e`. bcrypt cost 12, HS256 JWT (alg=none rejected), SHA-256 refresh hashing. Deps pinned for Go 1.22.
- **Phase 1 step 5 (store):** ✅ `internal/store` — `store.go` (handle, sentinels, users, refresh tokens), `catalog.go`, `cart.go`, `orders.go` (checkout txn with line snapshotting + cart clear; `TransitionOrder` with `FOR UPDATE` row lock → `domain.CanTransition` → audit event → loyalty earn on completed; ownership returns 404 not 403; idempotency-key NULL-on-empty). Builds + vets clean; **DB-backed tests deferred to Session 4** per plan.
- **Phase 1 step 6 (paystack):** ✅ `internal/paystack` — `Initialize`/`Verify` client (amount in pesewas, currency GHS, baseURL overridable) + `VerifySignature` (constant-time HMAC-SHA512). Tests green: signature valid/forged/wrong-secret/tampered/non-hex, plus Initialize/Verify via httptest.
- **Phase 1 step 7 (sse):** ✅ `internal/sse` — in-process per-order pub/sub broker; non-blocking publish, idempotent leak-free unsubscribe. Passes `-race`.
- **Phase 1 step 8 (httpapi):** ✅ committed `4f75b17`. `internal/httpapi` (10 files) — `server.go` route table + global chain (panic recovery → logging → CORS), `middleware.go` (Bearer/`?token=` auth, role gate, per-IP auth rate limit), handlers for auth/catalog/cart/checkout/orders/SSE/staff/admin, and `webhook_handlers.go` — the Paystack webhook, the **only** path to `paid`, enforcing TRD §5.2's four gates in order (signature → server-side verify → exact amount+GHS → legal transition), idempotent (paid→paid no-op), 5xx→retry / 200→stop, nil system actor. Standard error envelope throughout. The completing piece this session was the webhook handler (prior session stopped mid-step with it the one undefined symbol).
- **Phase 1 step 9 (cmd/api):** ✅ committed `9b8e778`. `cmd/api/main.go` — composition + lifecycle only: `config.LoadFromEnv` → `store.Open` + startup ping (fail fast) → `auth.NewTokenManager`, `paystack.NewClient`, `sse.NewBroker` → `httpapi.NewServer` → serve `Handler()`. `http.Server` with slowloris read timeouts but **`WriteTimeout: 0`** (a positive write deadline would sever live SSE streams); SIGINT/SIGTERM → bounded graceful `Shutdown`. Binary boots + fails fast on missing env.
- **Phase 1 step 10 (tests):** ✅ committed `8124f79` (store) + `013e4bb` (httpapi). Store suite against real Postgres — checkout (happy/empty/unavailable/duplicate-key), redemption (exact balance / over-balance reject / **concurrent double-spend → exactly one commit, balance 0, passes `-race`**), transition (legal/illegal/no-op/earn/refund-on-cancel). HTTP suite via `httptest` + fake Paystack — auth flow, wrong-password 401, ownership 404, staff 403 on manual `paid`, and the webhook four-gate matrix (happy/bad-sig/verify-failed/amount-mismatch/idempotent). Both skip without `TEST_DATABASE_URL`.
- **Phase 2 S2 (admin reports):** ✅ committed `b2ce07e`. `GET /admin/reports/summary?days=` (admin-only) — orders/day + revenue/day over a continuous (gap-filled) window; revenue counts only confirmed orders (not pending/cancelled). `store.ReportSummary` + handler + store & HTTP tests. Published to API brief §4.1.
- **Phase 2 S3 (password reset):** ✅ this session. Migration `0002_password_reset_tokens` (hash-only, single-use, 1h expiry; up/down verified, re-up clean). `auth.GenerateOpaqueToken`/`HashToken` (refresh helpers now delegate). `store.CreatePasswordResetToken` + `store.ConsumePasswordReset` (one txn: validate under `FOR UPDATE` → set password → clear user's reset tokens → revoke all refresh tokens). Endpoints `POST /auth/password-reset/{request,confirm}` (rate-limited, no enumeration, `invalid_token` for unknown/used/expired). **No email dep**: request logs the link server-side (`TODO(deploy)`) and never returns the token. Store + HTTP test suites added (request-is-generic, confirm happy/bad/expired/short-password, session-revocation). Published to API brief §2.9; recorded in DECISIONS.md. New error code `invalid_token`.
- **Pushed to remotes:** Backend repo `origin/main` at `4c867e7` (S3 password reset). Monorepo PR [#1](https://github.com/Manyle4/mug-e-store/pull/1) was last mirrored at `09c8ede` (`ffead51`) and is **behind** by both the admin-reports commit `b2ce07e` and the S3 commit `4c867e7` — re-mirror needed at next checkpoint.

## Next action

**Backend code + tests for Phase 1 are complete.** What remains is owner-only (cannot be done from this sandbox):
1. **Real-Paystack E2E (plan §4):** put a Paystack **test** secret key in `.env`, tunnel the local server (e.g. `ngrok http 8080`), register the webhook URL, run a real checkout with a test card, and confirm the order flips to `paid` over the live webhook + the SSE stream updates. Also verify a *pending* (not-success) payment does **not** flip the order.
2. **Deploy (plan §6):** host backend + Postgres; set `FRONTEND_ORIGIN`; switch the refresh cookie to `SameSite=None; Secure` if cross-origin.

If continuing in-sandbox instead: of the Phase 2 stretch list, **S1 (loyalty), S2 (admin reports), and S3 (password reset) are all done.** What's left is **S4 guest checkout** — but it's frontend-led and touches the cart/order ownership model (a public-API-shape change, so confirm the contract with the owner first). When starting any task: re-create the test DB if this box was reset (`createdb coffeemug_test` + apply **both** `migrations/0001_init.up.sql` and `migrations/0002_password_reset_tokens.up.sql`; DSN in the status block above), and re-mirror the monorepo PR at the end per the standing two-target rule.

**Two owner-only follow-ups (also from S3):** (1) wire a real email provider so the reset link is emailed instead of logged — replace the single `TODO(deploy)` log call in `handlePasswordResetRequest`; (2) the deploy step must run `0002` against prod Postgres.

**Monorepo PR — behind.** `Manyle4/mug-e-store` PR #1 last mirrored to standalone `09c8ede` (monorepo `ffead51`), `frontend/` untouched. It is now behind by the S2 and S3 commits. Re-run a mirror pass at the next checkpoint (clone, `git archive HEAD | tar -x -C backend/`, verify 0 `frontend/` files, push `backend-bootstrap`).

## Notes / open items

- **Two push targets** (see `DECISIONS.md` 2026-06-13):
  - `origin` = `https://github.com/deladei/coffemug-shop-backend-` — Go code at repo root. Local `main` is based on its existing "Initial commit" (README placeholder preserved).
  - `Manyle4/mug-e-store` — shared monorepo; backend code under `backend/`, delivered by **PR into `main`**, never touching `frontend/`.
- Specs live in `docs/` (moved there from repo root in Session 1).

# SESSION LOG

One short entry per session: what was built or changed.

## Session 1 — 2026-06-13

- Bootstrapped the repo. Verified tooling (Go 1.22.4, psql 18.1, git 2.51).
- Moved spec docs into `docs/`; added `.gitignore`, `.env.example`, and the working docs (`STATE.md`, `DECISIONS.md`, `SESSION_LOG.md`).
- Scaffolding commit `e83a81c` (Go module `coffeemug/backend`, git init, remote, doc stubs).
- `feat(db)` `129defd`: schema migration `0001_init`, verified up/down against Postgres.
- `feat(domain)` `890e349`: fulfilment-aware order state machine, exhaustive table-driven tests green.
- Decision: backend pushed to **two** repos — standalone `deladei/coffemug-shop-backend-` (root) and monorepo `Manyle4/mug-e-store` under `backend/` via PR (logged in `DECISIONS.md`).
- Pushed: backend repo `main` ← 3 commits; monorepo PR #1 opened (`backend-bootstrap` → `main`).

## Session 2 — 2026-06-13

- Resumed mid-Phase-1-step-8: `internal/httpapi` was written but failed to build — `server.go` referenced an undefined `handlePaystackWebhook` (prior session stopped right at the payment webhook).
- `feat(api)` `4f75b17`: completed the HTTP layer by adding `webhook_handlers.go` — the Paystack webhook enforcing TRD §5.2's four payment gates (signature → server-side verify → exact amount+GHS → legal transition), idempotent on retries, transient→5xx / permanent→200, system as nil actor. `go build`/`vet`/`test` all clean.
- `feat(cmd)` `9b8e778`: `cmd/api/main.go` entrypoint — wires config/store/auth/paystack/sse/httpapi into the single binary with a startup DB ping (fail fast), SSE-safe server timeouts (`WriteTimeout: 0`), and graceful SIGINT/SIGTERM shutdown. Verified the binary boots and fails fast on missing env.
- Pushed httpapi + cmd/api to `origin/main` (`703b8ab..9b8e778`). Monorepo PR #1 mirror still pending.
- `feat(loyalty)`: redemption at checkout (plan §3). Pure `domain.Redemption` rule (1 pt = 1 pesewa, capped at subtotal, surplus not spent, over-balance rejected) + 9-case table test; `store.Checkout` gained `RedeemPoints` — reads balance and writes the negative `redeem_at_checkout` ledger row inside the txn under a `SELECT … FOR UPDATE` lock on the user row (double-spend guard); `TransitionOrder` writes a compensating `refund_on_cancel` entry when a redeemed order is cancelled (covers `cancelDangling`). Checkout handler takes `points_to_redeem`, maps over-balance → `409 insufficient_points`. The discounted `total_pesewas` flows to Paystack and the webhook check unchanged → internally consistent.
- Contract change (CLAUDE.md §9): API consumption brief §2.5/§2.8 updated to add `points_to_redeem` + flip the "not yet a feature" note. Recorded in DECISIONS.md (field name, cap policy, refund-on-cancel, public-API-shape change). DB-backed redemption tests deferred to Session 4.
- Mirrored the monorepo PR #1 to standalone `3a9d3db` (monorepo `5df0579`) via clone + `git archive HEAD | tar -x -C backend/`; verified 0 `frontend/` files touched.
- `test(store)` `8124f79` + `test(httpapi)` `013e4bb`: the deferred DB-backed suite (plan §5). Created local `coffeemug_test` Postgres DB. Store tests (checkout, redemption incl. the concurrent double-spend race under `-race`, transition incl. refund-on-cancel) and HTTP-level tests (auth flow, ownership 404, staff 403 on manual paid, webhook four-gate matrix via a fake Paystack httptest server). Both skip without `TEST_DATABASE_URL`; the two DB suites share one DB so the tree runs with `go test -p 1 ./...`. 28 tests green.
- Re-mirrored the monorepo PR #1 twice as milestones landed (to `5df0579`, then `ffead51`), each verified 0 `frontend/` files.
- `feat(reports)` `b2ce07e`: Phase 2 S2 — `GET /admin/reports/summary?days=` (admin-only), orders/day + revenue/day over a continuous window; revenue excludes pending/cancelled. `store.ReportSummary` + handler + tests; published to API brief §4.1, recorded in DECISIONS.md.

## Session 3 — 2026-06-14

- Phase 2 **S3 password reset** (backend-only). Two endpoints: `POST /auth/password-reset/request {email}` (always `200`, generic body — no account enumeration) and `POST /auth/password-reset/confirm {token, password}`. Both rate-limited under the existing per-IP auth limiter.
- Migration `0002_password_reset_tokens` — a `refresh_tokens`-shaped table storing only the SHA-256 token hash; up/down verified, re-up clean. Applied to the local test DB.
- `internal/auth`: generalized the opaque-token primitives to `GenerateOpaqueToken`/`HashToken` (reused by reset); `GenerateRefreshToken`/`HashRefreshToken` now delegate to them so existing callers and tests are unchanged.
- `internal/store/password_reset.go`: `CreatePasswordResetToken` + `ConsumePasswordReset`. Consume is one transaction: look up the token `FOR UPDATE` (ErrNotFound / ErrTokenExpired) → update the user's password → delete all the user's reset tokens (single-use + invalidate older links) → delete all their refresh tokens (a password change revokes every session). New sentinel `ErrTokenExpired`.
- Handlers map unknown/used/expired tokens to one `400 invalid_token` (new error code); confirm enforces the same min-8-char rule as register. **No email dependency** (CLAUDE.md §5): the request handler logs the reset link server-side with a `TODO(deploy)` marker and never returns the token; the link is built from `FRONTEND_ORIGIN`.
- Tests: `internal/store/password_reset_test.go` (happy path incl. session revocation, single-use replay, unknown, expired-leaves-password-untouched) and `internal/httpapi/password_reset_test.go` (request generic for registered/unknown/malformed email, confirm happy path incl. old-pw-fails/new-pw-works + sessions revoked, bad-token, expired-token, short-password). Truncate lists in both test harnesses extended with `password_reset_tokens`. `go test -p 1 ./...` green with `TEST_DATABASE_URL`.
- Published to API brief §2.9; three entries added to DECISIONS.md (the feature/design, the no-email decision, the no-enumeration decision).

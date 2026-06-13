# STATE

Single source of "where we are" and "what's next". Read this at session start.

## Current status

- **Session:** 1 (in progress).
- **Build:** module sound (no Go packages yet).
- **Tests:** none yet.
- **Phase 0 (scaffolding):** ✅ committed `e83a81c`.
- **Phase 1 step 1 (schema migration):** ✅ committed `129defd`. Verified against Postgres (11 tables up, 0 down, re-up clean).
- **Phase 1 step 2 (domain state machine):** ✅ committed `890e349`. Exhaustive table-driven tests green (`internal/domain`).
- **Phase 1 step 3 (config loader):** ✅ committed `f48982f`. `internal/config` — env-only, refuses to boot without required secrets; tests green.
- **Phase 1 step 4 (auth):** ✅ committed `351c75e`. bcrypt cost 12, HS256 JWT (alg=none rejected), SHA-256 refresh hashing. Deps pinned for Go 1.22.
- **Phase 1 step 5 (store):** ✅ `internal/store` — `store.go` (handle, sentinels, users, refresh tokens), `catalog.go`, `cart.go`, `orders.go` (checkout txn with line snapshotting + cart clear; `TransitionOrder` with `FOR UPDATE` row lock → `domain.CanTransition` → audit event → loyalty earn on completed; ownership returns 404 not 403; idempotency-key NULL-on-empty). Builds + vets clean; **DB-backed tests deferred to Session 4** per plan.
- **Pushed to remotes:** Backend repo `origin/main` at `351c75e`; monorepo PR [#1](https://github.com/Manyle4/mug-e-store/pull/1) at `953bc9d`. `config`+`auth` pushed; **store commit is local, unpushed.**

## Next action

Phase 1 step 6: `internal/paystack` — initialize + verify client (POST /transaction/initialize, GET /transaction/verify/:ref against `PAYSTACK_BASE_URL`) and HMAC-SHA512 webhook signature verification. **Unit-test the signature check** against valid / forged / wrong-secret / tampered-body cases (no network). Commit `feat(paystack): init+verify client and webhook signature check`.

## Notes / open items

- **Two push targets** (see `DECISIONS.md` 2026-06-13):
  - `origin` = `https://github.com/deladei/coffemug-shop-backend-` — Go code at repo root. Local `main` is based on its existing "Initial commit" (README placeholder preserved).
  - `Manyle4/mug-e-store` — shared monorepo; backend code under `backend/`, delivered by **PR into `main`**, never touching `frontend/`.
- Specs live in `docs/` (moved there from repo root in Session 1).

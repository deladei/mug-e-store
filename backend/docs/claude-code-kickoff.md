# Claude Code — Kickoff Prompt (Session 1)

> **How to use this file:** open it, copy everything inside the code fence below, and paste it as your first message to Claude Code with this project folder open. `CLAUDE.md` and `/docs` must already be in the folder.

---

```
You are my backend pair-programmer for the Coffee Mug Shop project. I am the backend developer; the frontend is built by another team in a separate repo. We are starting from scratch — there is no code yet.

BEFORE WRITING ANY CODE:
1. Read CLAUDE.md in full. It is the working agreement and its rules are binding.
2. Read every file in /docs. Those specs are the source of truth. In particular:
   - docs/coffee-mug-shop-database-schema.md  -> build the migrations from this
   - docs/coffee-mug-shop-trd.md              -> architecture, stack, invariants
   - docs/coffee-mug-shop-api-consumption-brief.md -> the exact API contract
   - docs/coffee-mug-shop-user-journey.md     -> behaviour per endpoint
   - docs/coffee-mug-shop-implementation-plan.md -> build order and remaining work
Do not summarise them back to me. Build to them.

HARD RULES (from CLAUDE.md — repeated because they are non-negotiable):
- Money is integer pesewas everywhere. Never floats.
- An order becomes `paid` ONLY via a signature-verified Paystack webhook + a server-to-server verify call + amount/currency check. No client and no staff can set `paid` by hand.
- Strict layering: cmd -> internal/httpapi -> internal/store -> internal/domain. Business rules ONLY in internal/domain. SQL ONLY in internal/store.
- Module path is `coffeemug/backend`.
- No web framework, no ORM, no Redis. Go 1.22 stdlib routing + PostgreSQL + lib/pq.
- THE GIT LAW: every logical unit of work ends in a Conventional Commit; nothing is left uncommitted; push to origin at the end of the session. Backend code goes to MY repo only — never to the frontend repo. Never commit .env or any secret.

PHASE 0 — repo and tooling (do this first, then stop and show me the result):
1. Confirm `go version` is 1.22+ and `psql` is available. If not, tell me what to install and stop.
2. Create `.gitignore` (ignore: .env, /bin/, the built binaries, *.test, *.out, editor/OS files) and a `.env.example` with placeholder values for: DATABASE_URL, JWT_SECRET, PAYSTACK_SECRET_KEY, PAYSTACK_BASE_URL, PORT, DELIVERY_FEE_PESEWAS, FRONTEND_ORIGIN.
3. `git init`, set default branch to `main`, add remote:
     origin = https://github.com/deladei/coffemug-shop-backend-
   Run `git ls-remote origin` to confirm access and whether the repo already has commits. If it is non-empty or access fails, STOP and tell me — do not force anything.
4. Initialise the Go module: `go mod init coffeemug/backend`.
5. Create STATE.md, DECISIONS.md, SESSION_LOG.md (stubs are fine).
6. Commit: `chore: initialize Go module, repo scaffolding, and working docs`. Do NOT push yet. Show me the file tree and the commit, then continue.

PHASE 1 — build in dependency order. ONE commit per layer. Each layer must `go build ./...` and `go vet ./...` clean before you commit it; layers with tests must pass `go test ./...`.
  1. migrations/0001_init.up.sql + .down.sql, exactly matching the schema doc (all tables, constraints, indexes). Commit: `feat(db): initial schema migration`.
  2. internal/domain — the order status type, the fulfilment-aware transition function, terminality, status validation. WRITE THE TABLE-DRIVEN TESTS FIRST, then the code, covering every legal and illegal transition for both fulfilment types. Commit: `feat(domain): order state machine with tests`.
  3. internal/config — env-only config loader that refuses to start without the required secrets. Commit.
  4. internal/auth — bcrypt (cost 12), HS256 JWT access tokens, refresh-token generation + SHA-256 hashing. Commit.
  5. internal/store — store.go (db handle, error sentinels, users, refresh tokens), catalog.go, cart.go, orders.go (checkout as a single transaction with line snapshotting + cart clear; transitions with a row lock, audit event, and loyalty earn on completion). Commit: `feat(store): persistence layer`.
  6. internal/paystack — initialize + verify client, and HMAC-SHA512 webhook signature verification. Unit-test the signature check against forged / valid / wrong-secret / tampered-body cases (no network). Commit.
  7. internal/sse — in-process pub/sub broker for order-status events, with leak-free subscriber cleanup. Commit.
  8. internal/httpapi — server.go (Go 1.22 routing, the full route table from the API brief, middleware: auth via Bearer or ?token= for SSE, role gate, CORS to FRONTEND_ORIGIN, slog logging, panic recovery, per-IP rate limit on auth) + the handlers (auth, catalog, cart, checkout, orders, SSE, the Paystack webhook with all four gates, and the staff/admin endpoints). Commit: `feat(api): http layer and handlers`.
  9. cmd/api/main.go (graceful shutdown, no write timeout so SSE survives), cmd/seeduser (role-seeding helper), Makefile (run/build/test/migrate-up/migrate-down/seed), seed.sql (the demo menu in pesewas). Commit.

PHASE 2 — make it run locally and prove it:
  1. Tell me the exact commands to create a local Postgres DB/user, then have me create `.env` from `.env.example` (I will paste in my Paystack TEST secret key myself — never ask me to commit it).
  2. Run `make migrate-up`, `make seed`, `make run`. Confirm `curl localhost:8080/api/v1/healthz` returns ok. Fix anything that fails and commit the fixes.
  3. Update STATE.md and SESSION_LOG.md. Commit: `docs: session 1 status`. Then PUSH everything to origin.

SESSION 1 IS DONE WHEN: the repo builds, domain + paystack tests are green, the server starts and answers healthz, and every commit is pushed to origin. A partial-but-building, fully-committed state is an acceptable stopping point if we run low on time. An uncommitted state is NOT.

DO NOT, in session 1: build loyalty redemption, attempt a live end-to-end Paystack payment, or touch the frontend repo. Those are later sessions (see the implementation plan). If you finish early, stop and ask me before starting them.

ASK ME BEFORE: changing the public API shape, deviating from any spec (and log it in DECISIONS.md), or adding any dependency.
```

---

## Sessions after this one (for your reference — don't paste these yet)

Run them as separate Claude Code sessions, each ending under the same Git Law:

- **Session 2 — real Paystack end-to-end.** Use your Paystack test keys + an `ngrok` tunnel so Paystack can reach your webhook; pay with a test card; confirm the order flips to `paid` via the webhook and your SSE stream receives the event. This is the highest-risk integration — do it before anything cosmetic.
- **Session 3 — loyalty redemption at checkout.** The reserved money-path feature. Implement inside the checkout transaction; write the concurrent double-spend test (two redemptions of the same balance — only one may succeed). Spec is in the implementation plan.
- **Session 4 — fill the test suite.** Store tests against a test DB; HTTP tests with `httptest` (auth flow, ownership 404, staff-cannot-set-paid 403).
- **Session 5 — deploy** (managed Postgres + the single binary; set FRONTEND_ORIGIN; switch the refresh cookie to SameSite=None; Secure for cross-origin prod).
- **Contract hand-off (any time the API changes):** open a pull request adding/updating the API contract doc in the frontend repo `Manyle4/mug-e-store`. If you are not a collaborator there, fork and PR. Never push backend code there.

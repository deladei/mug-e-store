# CLAUDE.md — Coffee Mug Shop Backend

This file is the working agreement for every Claude Code session in this repo. Read it before writing or changing any code. If anything here conflicts with a request, surface the conflict instead of silently overriding this file.

---

## ▶ SESSION START PROTOCOL (do this automatically, every session)

When you open this folder, before anything else:

1. **Read this entire file** (the rules below are binding).
2. **Read `STATE.md`** — it holds the current status and the single **Next action**.
3. **Skim the `/docs` specs relevant to that Next action** (they are the source of truth — see §2).
4. **Begin the Next action immediately.** Do not wait for a detailed brief; a one-word "go" from me is full authorization to start the Next action in `STATE.md`. Work through it under the Git Law (§3) and Definition of Done (§6).
5. **Obey the "Ask me before" list (§9)** — stop and ask only for those cases.
6. **At the end of the session**, update `STATE.md` (status + the new Next action), append a line to `SESSION_LOG.md`, commit, and push.

If `STATE.md` is missing or has no Next action, ask me what to work on. Otherwise, proceed.

---

## 1. What this is

The backend service for **Coffee Mug Shop** — an online ordering app for a single coffee shop in Accra, Ghana (browse → order → pay → prepare → fulfil, with pickup/delivery, Paystack payment, real-time status, and loyalty points).

- **This repo (backend, mine):** `https://github.com/deladei/coffemug-shop-backend-`
- **Frontend team's repo (not mine):** `https://github.com/Manyle4/mug-e-store`
- I own the backend only. The Next.js frontend is built by other team members.

## 2. Source of truth

The specs in `/docs` define the system. Build the code to match them; do not invent behaviour that contradicts them.

- `docs/coffee-mug-shop-prd.md` — product requirements, scope, phasing
- `docs/coffee-mug-shop-trd.md` — architecture, stack, non-functional requirements
- `docs/coffee-mug-shop-database-schema.md` — the schema (build migrations from this)
- `docs/coffee-mug-shop-user-journey.md` — each step mapped to its endpoint
- `docs/coffee-mug-shop-api-consumption-brief.md` — the API contract the frontend codes against
- `docs/coffee-mug-shop-implementation-plan.md` — build order and remaining work
- `docs/claude-code-kickoff.md` — the detailed build playbook (optional manual reference; `STATE.md` is the live driver)
- `docs/coffee-mug-shop-wireframe-brief.md` — (frontend reference)

If code and a spec disagree, that is a bug. Fix one of them **deliberately** and record the decision in `DECISIONS.md`. Never let them drift silently.

## 3. The Git Law (non-negotiable)

1. **Every logical unit of work ends in a commit.** No session ever ends with uncommitted work.
2. **Conventional Commits**: `feat:`, `fix:`, `chore:`, `test:`, `docs:`, `refactor:`. The body explains *why*, not just *what* — this history is defended in a viva.
3. **Push to `origin` at the end of every session** and after every milestone.
4. **Backend code goes to two targets** (overrides the original "this repo only" rule — see `DECISIONS.md` 2026-06-13): (a) the standalone backend repo `deladei/coffemug-shop-backend-` with the Go code at the repo root; (b) the shared monorepo `Manyle4/mug-e-store` under a **`backend/`** subfolder, alongside the existing `frontend/`. Delivery into the monorepo is by **pull request into `main`** — never a force-push, never overwriting or touching `frontend/`.
5. **The API contract** (`docs/coffee-mug-shop-api-consumption-brief.md`) is published to the monorepo via the same pull-request flow whenever it changes — never bundled in a way that overwrites frontend files.
6. **Never commit secrets.** `.env` is git-ignored. Only `.env.example` (with placeholder values) is tracked. If a real key is ever staged, stop and remove it before committing.

## 4. Architecture invariants (must never be violated)

These are the rules the specs encode. They are the reason the system is correct; do not "simplify" them away.

- **Money is integer pesewas** (1 GHS = 100 pesewas), `BIGINT` in SQL, `int64` in Go. No floating-point money anywhere — not in storage, transport, or calculation.
- **Payment truth comes only from Paystack.** An order becomes `paid` only when a webhook passes all four gates: valid HMAC-SHA512 signature → server-to-server verify call returns success → verified amount equals the stored order total and currency is GHS → the transition is legal. No client and no staff member can set an order to `paid` by hand.
- **Layering is strict:** `cmd/` → `internal/httpapi` → `internal/store` → `internal/domain`. A layer may only call the layer below it. **All business rules live in `internal/domain`. Only `internal/store` writes SQL.** Handlers translate HTTP to function calls and nothing more.
- **Idempotency:** `orders.paystack_reference` is `UNIQUE`; transitioning an order to the status it already holds is a successful no-op (this is what makes webhook retries safe); checkout accepts a client `idempotency_key` stored `UNIQUE`.
- **The order lifecycle is a fulfilment-aware finite state machine** with one source of truth in `internal/domain`. Pickup orders never enter `out_for_delivery`; delivery orders never reach `completed` without shipping first. Illegal jumps are rejected.
- **Order history is immutable:** `order_lines` snapshot item name, variant name, and unit price at checkout. Editing the menu later never changes a past order.
- **Loyalty is an append-only ledger:** balance is always `SUM(delta)`; there is no stored balance column. Points are earned on the `completed` transition only.
- **Availability is enforced server-side**, not just in the UI.
- **Security:** bcrypt cost 12; JWT HS256 access tokens (15 min, `uid`+`role` claims, signing method asserted on parse); refresh tokens stored only as SHA-256 hashes, single-use with rotation, delivered as an `httpOnly` cookie; identical errors for unknown-email vs wrong-password (no enumeration); order ownership failures return `404`, never `403`; rate limit on auth endpoints.

## 5. Stack (pinned — do not add to without asking)

- Go 1.22+ (stdlib `net/http` method+path routing — **no web framework**)
- PostgreSQL (via `github.com/lib/pq` — **no ORM**)
- `github.com/golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`
- Real-time via **in-process** SSE broker — **no Redis, no message queue**
- Module path: `coffeemug/backend`

Adding any framework, ORM, Redis, or new dependency requires explicit approval and a `DECISIONS.md` entry.

## 6. Definition of Done (every task)

A task is done only when **all** of these hold:
1. `go build ./...` succeeds.
2. `go vet ./...` is clean.
3. `go test ./...` passes.
4. Work is committed with a Conventional Commit message.
5. Committed work is pushed (at session end at the latest).

## 7. Working docs to maintain (commit them)

- `STATE.md` — current status + the single Next action. Update at the end of each session. **This is what the Session Start Protocol reads to know what to do.**
- `DECISIONS.md` — every deviation from a spec or pinned choice, with the reason.
- `SESSION_LOG.md` — one short entry per session: what was built/changed.

## 8. Authorship clause

I must be able to explain every file in this repo in a viva. Review each diff before it is committed. For any non-obvious choice, put the reasoning in the commit body or in `DECISIONS.md`. The goal is a repo I authored with an AI pair, not a repo I received.

## 9. Ask me before

- Changing the **public API shape** (the frontend codes against it — a change there is a coordination event).
- **Deviating from any spec** (record it in `DECISIONS.md`).
- **Adding a dependency** (see §5).
- Anything touching **money or the payment path** in a way not already described in the specs.

# Production Readiness & Security Posture

This document triages the standard production / security checklist against **this
system's actual scope**: a single-tenant backend for one coffee shop in Accra,
graded and defended in a viva. It records what is implemented, what is
deliberately out of scope (and why), and what is an owner/operator step. It is an
ADR-style companion to `DECISIONS.md` — point at it when a reviewer asks "did you
think about X?"

The guiding principle is **fit to scope**. Adding circuit breakers, multi-region
DR, or chaos engineering to a single-shop ordering API would be cargo-culting; a
senior engineer's job is to apply the controls that match the real risk, and to
say plainly which ones don't.

Legend: ✅ implemented · 🧭 deliberate non-goal · 🛠 owner/operator step

---

## Security

| Concern | Status | Where / Notes |
|---|---|---|
| Input sanitization & injection prevention | ✅ | All SQL is parameterized (`lib/pq`, no string-built queries, no ORM). JSON bodies are size-capped at 1 MiB (`http.MaxBytesReader`) and reject unknown fields (`DisallowUnknownFields`). Money is parsed as `int64`, never `eval`-style. |
| Authentication | ✅ | bcrypt cost 12; HS256 JWT access tokens with the signing method asserted on parse (`alg=none` rejected). |
| Authorization, roles & permissions | ✅ | `requireAuth` / `requireRole` middleware; customer/staff/admin tiers; order-ownership failures return `404`, never `403` (no existence leak). |
| Session management & token expiry | ✅ | 15-minute access tokens; refresh tokens are opaque, **SHA-256-hashed at rest**, single-use with rotation, delivered as an `httpOnly` cookie; a password reset revokes all of a user's refresh tokens. |
| Secrets management | ✅ / 🛠 | Config is env-only; `.env` is gitignored; the server **fails fast** if a required secret is missing. 🛠 In production, inject secrets via the platform's secret store (Render/Fly env), never a committed file. |
| HTTP security headers | ✅ | `securityHeaders` middleware: `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, restrictive CSP (`default-src 'none'`), CORP, and HSTS. |
| HTTPS / TLS / certificate rotation | 🛠 | TLS is **terminated at the platform edge** (Render/Fly/Cloudflare), which also auto-renews certificates. The app speaks HTTP behind that proxy and emits HSTS so the browser enforces TLS. No in-process cert handling by design. |
| Rate limiting & abuse prevention | ✅ / 🛠 | Per-IP rate limiter on all credential endpoints (register/login/guest/password-reset). Body-size caps blunt payload-flood abuse. 🛠 Broad L7 rate limiting / WAF is best done at the edge (Cloudflare) rather than per-instance, especially with multiple replicas; deferred there intentionally. |
| Dependency scanning & vulnerability patching | ✅ | `govulncheck` runs in CI on every push/PR. The dependency surface is tiny and pinned (jwt, bcrypt, lib/pq). Stdlib CVEs are patched by building on a current Go toolchain (CI + the Docker base track the latest 1.22.x). |
| Audit trails / tamper-evident logging | ✅ / 🧭 | Every order state change writes an append-only `order_events` row (actor, from→to, timestamp) — the order history is immutable, and `order_lines` snapshot name+price at checkout. Cryptographic tamper-evidence (hash chaining / WORM storage) is 🧭 out of scope for this domain. |
| Concurrency & race-condition prevention | ✅ | Balance reads and order transitions happen inside a transaction under `SELECT … FOR UPDATE`; the loyalty double-spend test runs under `-race` and asserts exactly one of two concurrent redemptions commits. |
| PII handling, retention & deletion | ✅ / 🛠 | PII collected is minimal (name, email, phone). Passwords are never stored in the clear; password hashes are never serialized to clients. 🛠 A formal retention window and a "delete my account" path are operator policy — the data model supports deletion (FKs cascade), but the endpoint/runbook is a deploy-time decision. |
| Regulatory compliance (GDPR / HIPAA) | 🧭 / 🛠 | **HIPAA is N/A** — no health data. GDPR-style data-subject rights (export/delete) are 🛠 operator policy if EU users are in scope; the minimal-PII model and cascade-deletable schema make them straightforward to add. |
| Multi-tenancy & data isolation | 🧭 | **Explicit non-goal** (PRD §2): single shop, single menu, single admin team. No tenant column, no per-tenant RLS — adding them would be designing for a requirement that does not exist. |

## Testing & quality

| Concern | Status | Where / Notes |
|---|---|---|
| Unit tests | ✅ | Domain state machine (exhaustive table tests), auth, config, Paystack signature, SSE broker (`-race`). |
| Integration tests | ✅ | DB-backed `store` suite against real Postgres (checkout, redemption, transitions). |
| End-to-end / HTTP tests | ✅ | `httptest`-driven API suite with a fake Paystack: auth flow, ownership 404, staff 403 on manual `paid`, and the webhook four-gate matrix. |
| Regression tests | ✅ | Each shipped feature (loyalty, reset, guest, reports) added tests that pin its rules; they run on every push. 🛠 Real Paystack live-E2E is the one regression that needs real keys + a tunnel (owner step). |
| Coverage thresholds enforced in CI | ✅ | CI computes total coverage and fails below the configured floor (see `.github/workflows/ci.yml`). |
| Code review process & standards | ✅ | `CLAUDE.md` is the working agreement (Git Law, Definition of Done, authorship clause); backend lands in the shared monorepo **only by pull request** — never a direct push to `main`. CI gates every PR. |
| Load & stress testing | 🧭 / 🛠 | Out of scope for the grade. Realistic shop load is tens of concurrent users; the in-process SSE broker and a single Postgres comfortably cover it. A `k6`/`vegeta` smoke against the deployed URL is a cheap 🛠 follow-up if needed. |
| Chaos engineering & resilience testing | 🧭 | Out of scope: one service, one DB, no microservice blast radius to inject failures into. The meaningful resilience properties (graceful shutdown, fail-fast on bad config, idempotent webhook retries) are covered by design + tests. |

## Reliability & operations

| Concern | Status | Where / Notes |
|---|---|---|
| Error handling & graceful degradation | ✅ | One standard JSON error envelope; panic-recovery middleware; `/healthz` is a real DB-backed liveness probe; the server boots only after a successful startup ping. |
| Retry logic with backoff & idempotency | ✅ / 🧭 | The system is built to be **safely retryable**: `orders.paystack_reference` is `UNIQUE`, checkout takes a client `idempotency_key`, and transitioning to the current status is a successful no-op — so Paystack webhook retries and client retries are safe. Client-side exponential backoff is the caller's concern; in-app backoff loops are 🧭 unnecessary given idempotency. |
| Circuit breakers & fallback behavior | 🧭 | One external dependency (Paystack), called server-to-server only on the webhook path. A breaker adds failure modes without reducing real risk here; the webhook simply returns 5xx so Paystack retries. |
| Caching strategy & invalidation | 🧭 | No cache layer by design. The hot read (the menu) is a small, indexed query; introducing a cache would add an invalidation-correctness problem (stale prices) for no measured benefit. Prices are always resolved live at cart/checkout time precisely to avoid stale-figure bugs. |
| RTO / RPO | 🛠 | Defined as operator targets: **RPO** = the managed Postgres backup/PITR window (provider default, e.g. continuous/24h); **RTO** = redeploy the stateless binary (minutes) + restore the DB from the latest backup. The app holds no durable state outside Postgres, so recovery is "point DB at a restore and restart." |
| Disaster recovery plan | 🛠 | Stateless service + managed Postgres → DR is: (1) restore Postgres from backup, (2) redeploy the image, (3) re-point `DATABASE_URL`, (4) run `make migrate-up` if needed. No bespoke state to reconstruct. |
| Architecture diagrams & ADRs | ✅ | Strict layering documented in `CLAUDE.md §4` and `docs/architecture.md`; every deviation/decision is an ADR row in `DECISIONS.md`. |
| Accessibility | 🧭 | A backend (JSON API) concern only insofar as it returns clean, structured data and human-readable messages; visual/interaction accessibility lives in the frontend team's repo. |

---

## Summary

The controls that match a single-tenant payment-handling ordering API are
**implemented and tested**: parameterized data access, layered authz with
no-enumeration semantics, hashed rotating sessions, security headers, per-IP
auth rate limiting, an append-only audit trail, transaction-level concurrency
safety, idempotent payment handling, and CI with dependency scanning + a
coverage gate.

The items marked 🧭 are **deliberate non-goals** justified by scope, not
oversights. The items marked 🛠 are **operator steps** that need real
infrastructure (a host, real secrets, an edge/WAF, a backup policy) and cannot
be exercised from the build sandbox.

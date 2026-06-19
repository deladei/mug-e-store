# Coffee Mug Shop — Backend

Backend service for **Coffee Mug Shop**, an online ordering app for a single
coffee shop in Accra, Ghana: browse → order → pay → prepare → fulfil, with
pickup/delivery, Paystack payment, real-time order status (SSE), and loyalty
points.

Go 1.22+ (stdlib `net/http`, no framework), PostgreSQL (`lib/pq`, no ORM),
in-process SSE broker. Money is **integer pesewas** everywhere (1 GHS = 100
pesewas). The order lifecycle and all business rules live in `internal/domain`;
only `internal/store` writes SQL. See `CLAUDE.md` for the full working agreement
and `docs/` for the specs.

## Run it locally

Requires Go 1.22+, a running PostgreSQL, and the `psql` client on PATH.

```sh
# 1. Configure
cp .env.example .env          # then fill in DATABASE_URL, JWT_SECRET, Paystack test key

# 2. Create the database (example; match DATABASE_URL)
createdb coffeemug

# 3. Load env into the shell (so make/go see DATABASE_URL etc.)
set -a; . ./.env; set +a

# 4. Schema + demo data
make migrate-up               # applies migrations/*.up.sql in order
make seed                     # demo menu + admin@coffeemug.shop / staff@coffeemug.shop (password123)

# 5. Run
make run                      # listens on :$PORT (default 8080)
curl localhost:8080/api/v1/healthz   # -> {"status":"ok"}
```

`make help` lists every target. `make build` produces `bin/api` and `bin/seed`.

## Test

```sh
make test                     # go test -p 1 ./...
```

The store and HTTP suites are DB-backed and **skip** unless `TEST_DATABASE_URL`
is set; without it `make test` still passes (the DB suites are simply skipped).
With a Postgres test database:

```sh
export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/coffeemug_test?sslmode=disable"
make test
```

Run the suite serially (`-p 1`): the store and httpapi packages share one test
database and truncate between tests, so running them in parallel is not safe.

## Deployment

A single Go binary plus a managed Postgres. Build `bin/api`, set the environment
variables (`DATABASE_URL`, `JWT_SECRET`, `PAYSTACK_SECRET_KEY`, `FRONTEND_ORIGIN`,
…), run `make migrate-up` once against the production database, then start the
binary. If the frontend is on a different origin, the refresh cookie must be
switched to `SameSite=None; Secure`.

### Render (Blueprint)

`render.yaml` provisions the whole stack. In the Render dashboard: **New →
Blueprint →** pick this repo. Render reads the blueprint, creates the managed
Postgres and the Docker web service, wires `DATABASE_URL` between them, and
generates `JWT_SECRET`. Then, in the service's **Environment** tab, set the two
dashboard-only secrets — `PAYSTACK_SECRET_KEY` (`sk_test_…`/`sk_live_…`) and
`FRONTEND_ORIGIN` (the deployed frontend URL) — and deploy. The container builds
from the `Dockerfile`; migrations run automatically via `preDeployCommand` on
paid plans (on the free plan, run that same psql loop once from the Render
Shell). Health check: `GET /api/v1/healthz`.

Two caveats called out in the blueprint comments: Render's **free** Postgres
expires after ~90 days (use `basic-256mb`+ for anything real), and free web
services sleep on inactivity and cannot run `preDeployCommand`.

**Deploy on merge.** `render.yaml` sets `autoDeploy: false`; production ships only
via the CI-gated Deploy workflow (`.github/workflows/deploy.yml`), which fires a
Render Deploy Hook **after CI passes on `main`**. One-time setup: Render → the
`coffeemug-api` service → Settings → **Deploy Hook** → copy the URL → add it as
the repo secret `RENDER_DEPLOY_HOOK_URL` (Settings → Secrets and variables →
Actions). Until that secret is set, the Deploy workflow fails fast with a clear
message. (To skip CI gating, flip `autoDeploy: true` and disable the workflow.)

## Payment integrity

An order becomes `paid` **only** through the Paystack webhook, which must pass
four gates in order: valid HMAC-SHA512 signature → server-to-server verify
returns success → verified amount equals the stored order total and currency is
GHS → the transition is legal. No client and no staff member can set an order to
`paid` by hand.

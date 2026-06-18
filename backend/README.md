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

## Payment integrity

An order becomes `paid` **only** through the Paystack webhook, which must pass
four gates in order: valid HMAC-SHA512 signature → server-to-server verify
returns success → verified amount equals the stored order total and currency is
GHS → the transition is legal. No client and no staff member can set an order to
`paid` by hand.

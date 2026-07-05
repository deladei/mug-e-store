# Coffee Mug Shop backend — developer tasks.
#
# Most targets read DATABASE_URL from the environment. The simplest way is to
# copy .env.example to .env, fill it in, and `set -a; . ./.env; set +a` before
# running make — or export DATABASE_URL inline. The migrate targets shell out to
# `psql`, so the Postgres client must be on PATH.

# Apply the schema migrations in filename order. Stops on the first error.
MIGRATIONS := $(sort $(wildcard migrations/*.up.sql))
ROLLBACKS  := $(sort $(wildcard migrations/*.down.sql))

.PHONY: build run seed test vet migrate-up migrate-down help

help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'

build: ## Compile the API and seed binaries into ./bin.
	go build -o bin/api ./cmd/api
	go build -o bin/seed ./cmd/seed

run: ## Run the API server (needs a full .env).
	go run ./cmd/api

seed: ## Load the demo menu + admin/staff accounts (needs DATABASE_URL).
	go run ./cmd/seed

test: ## Run the full test suite serially (set TEST_DATABASE_URL for DB tests).
	go test -p 1 ./...

vet: ## Static analysis.
	go vet ./...

migrate-up: ## Apply all up migrations in order (needs psql + DATABASE_URL).
	@test -n "$(DATABASE_URL)" || { echo "DATABASE_URL is not set"; exit 1; }
	@for f in $(MIGRATIONS); do \
		echo "applying $$f"; \
		psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -q -f "$$f" || exit 1; \
	done

migrate-down: ## Roll back all down migrations in reverse order (DESTRUCTIVE).
	@test -n "$(DATABASE_URL)" || { echo "DATABASE_URL is not set"; exit 1; }
	@for f in $$(echo $(ROLLBACKS) | tr ' ' '\n' | sort -r); do \
		echo "reverting $$f"; \
		psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -q -f "$$f" || exit 1; \
	done

# Multi-stage build for the Coffee Mug Shop backend. The result is a small
# Alpine image holding two static (CGO-disabled) binaries — the API server and
# the demo seeder — plus the SQL migrations and the psql client, so a single
# image can migrate, seed, and serve. Works on any container host (Render,
# Railway, Fly.io, plain Docker).

# ---- build stage ----
# Pinned to a specific patched 1.22.x (not the floating 1.22 tag) for
# reproducible, auditable builds — this patch carries the stdlib security fixes
# that govulncheck flags on older toolchains.
FROM golang:1.22.12-alpine AS build
WORKDIR /src

# Download modules first so this layer caches unless go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Pure-Go build (lib/pq, jwt and bcrypt are all pure Go): CGO off yields a
# static binary that runs on a bare Alpine without libc surprises.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api  ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---- runtime stage ----
FROM alpine:3.20
WORKDIR /app

# ca-certificates: outbound HTTPS to Paystack. postgresql-client: the migrate
# step shells out to psql. wget (busybox) backs the healthcheck.
RUN apk add --no-cache ca-certificates postgresql-client && \
    adduser -D -u 10001 app

COPY --from=build /out/api  /app/api
COPY --from=build /out/seed /app/seed
COPY migrations /app/migrations

USER app
EXPOSE 8080

# The API exposes a DB-backed liveness probe; hit it from inside the container.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/api/v1/healthz" || exit 1

CMD ["/app/api"]

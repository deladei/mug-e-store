-- 0001_init.up.sql
-- Coffee Mug Shop — initial schema.
-- Engine: PostgreSQL. Money is BIGINT pesewas (1 GHS = 100 pesewas); no floats.
-- Built from docs/coffee-mug-shop-database-schema.md (v1.0). Eleven tables,
-- four groups: identity, catalog, cart, orders + ledger.

BEGIN;

-- ============================================================
-- Identity
-- ============================================================

CREATE TABLE users (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL UNIQUE,
    phone         TEXT        NOT NULL DEFAULT '',
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'customer'
                              CHECK (role IN ('customer', 'staff', 'admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,  -- SHA-256 of the token, never the token
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Catalog
-- ============================================================

CREATE TABLE categories (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT   NOT NULL UNIQUE,
    sort_order INT    NOT NULL DEFAULT 0
);

CREATE TABLE items (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category_id  BIGINT      NOT NULL REFERENCES categories (id),
    name         TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    image_url    TEXT        NOT NULL DEFAULT '',
    is_available BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Price lives on the variant, not the item: a "Latte" has no single price.
CREATE TABLE item_variants (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    item_id       BIGINT NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    name          TEXT   NOT NULL,
    price_pesewas BIGINT NOT NULL CHECK (price_pesewas >= 0),
    sort_order    INT    NOT NULL DEFAULT 0
);

-- ============================================================
-- Cart
-- ============================================================

CREATE TABLE carts (
    id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE
);

-- The cart stores no price; it is always resolved against the live menu.
CREATE TABLE cart_lines (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id         BIGINT NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    item_variant_id BIGINT NOT NULL REFERENCES item_variants (id) ON DELETE CASCADE,
    quantity        INT    NOT NULL CHECK (quantity > 0),
    UNIQUE (cart_id, item_variant_id)
);

-- ============================================================
-- Orders and ledger
-- ============================================================

CREATE TABLE orders (
    id                   BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id              BIGINT      NOT NULL REFERENCES users (id),
    status               TEXT        NOT NULL DEFAULT 'pending_payment'
                                     CHECK (status IN (
                                         'pending_payment', 'paid', 'preparing',
                                         'ready', 'out_for_delivery', 'completed',
                                         'cancelled'
                                     )),
    fulfilment           TEXT        NOT NULL
                                     CHECK (fulfilment IN ('pickup', 'delivery')),
    address              TEXT        NOT NULL DEFAULT '',
    phone                TEXT        NOT NULL DEFAULT '',
    subtotal_pesewas     BIGINT      NOT NULL,
    delivery_fee_pesewas BIGINT      NOT NULL DEFAULT 0,
    discount_pesewas     BIGINT      NOT NULL DEFAULT 0,
    total_pesewas        BIGINT      NOT NULL,
    paystack_reference   TEXT        UNIQUE,
    idempotency_key      TEXT        UNIQUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_status ON orders (status);  -- powers the staff queue
CREATE INDEX idx_orders_user   ON orders (user_id); -- powers customer history

-- Immutable snapshot of what was bought: names and price are COPIED, not FKs,
-- so re-pricing or deleting a menu item never alters a past order.
CREATE TABLE order_lines (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id           BIGINT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    item_name          TEXT   NOT NULL,
    variant_name       TEXT   NOT NULL,
    unit_price_pesewas BIGINT NOT NULL,
    quantity           INT    NOT NULL
);

-- Audit trail of the state machine. actor_id is NULL when the system (the
-- payment webhook) is the actor rather than a staff member.
CREATE TABLE order_events (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id    BIGINT      NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    from_status TEXT        NOT NULL,
    to_status   TEXT        NOT NULL,
    actor_id    BIGINT      REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only ledger: balance is SUM(delta), never a stored column.
-- Positive delta = earned, negative = redeemed (redemption not yet built).
CREATE TABLE loyalty_ledger (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id),
    order_id   BIGINT      REFERENCES orders (id),
    delta      BIGINT      NOT NULL,
    reason     TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;

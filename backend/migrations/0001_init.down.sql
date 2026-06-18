-- 0001_init.down.sql
-- Clean teardown of the initial schema. Tables are dropped in reverse
-- dependency order so foreign keys never block the drop.

BEGIN;

DROP TABLE IF EXISTS loyalty_ledger;
DROP TABLE IF EXISTS order_events;
DROP TABLE IF EXISTS order_lines;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS cart_lines;
DROP TABLE IF EXISTS carts;
DROP TABLE IF EXISTS item_variants;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

COMMIT;

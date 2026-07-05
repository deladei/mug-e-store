-- 0003_users_is_guest.down.sql
-- Reverse 0003. Dropping the column also drops the guest marker; any guest rows
-- left behind become indistinguishable from registered users (harmless — they
-- still cannot log in, their password hash is unusable).

BEGIN;

ALTER TABLE users DROP COLUMN is_guest;

COMMIT;

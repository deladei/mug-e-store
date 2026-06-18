-- 0002_password_reset_tokens.down.sql
-- Reverses 0002: drops the password-reset token table.

BEGIN;

DROP TABLE IF EXISTS password_reset_tokens;

COMMIT;

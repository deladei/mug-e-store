-- 0002_password_reset_tokens.up.sql
-- Phase 2 S3 — password reset. A reset link carries an opaque token; like the
-- refresh-token table, only the SHA-256 hash is stored, never the raw token, so
-- a leak of this table yields no usable reset links. Tokens are short-lived
-- (expires_at) and single-use (deleted when consumed). The whole row set for a
-- user is cleared on a successful reset, invalidating any older outstanding links.

BEGIN;

CREATE TABLE password_reset_tokens (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,  -- SHA-256 of the token, never the token
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A user may have several outstanding tokens; we look them up / clear them by user.
CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);

COMMIT;

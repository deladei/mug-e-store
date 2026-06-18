-- 0003_users_is_guest.up.sql
-- Guest checkout (PRD S4). A guest is a real but passwordless users row, so the
-- entire user_id-keyed system (cart, orders, ownership, SSE) keeps working
-- unchanged — a guest simply IS a customer who cannot log in. This flag marks
-- such rows so loyalty earning can be suppressed for them (a guest can never log
-- back in to spend points) and so they can be told apart from registered users.
-- See DECISIONS.md (2026-06-16) for why this model was chosen over a nullable
-- orders.user_id.

BEGIN;

ALTER TABLE users ADD COLUMN is_guest BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;

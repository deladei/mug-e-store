package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CreatePasswordResetToken stores a new reset token's hash and expiry for a
// user. Only the hash reaches the database (see auth.HashToken). A user may hold
// several outstanding tokens; the newest simply adds a row, and all of a user's
// rows are cleared the moment any one of them is consumed (see ConsumePasswordReset).
func (s *Store) CreatePasswordResetToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	const q = `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`
	if _, err := s.db.ExecContext(ctx, q, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("store: creating password reset token: %w", err)
	}
	return nil
}

// ConsumePasswordReset atomically redeems a reset token and sets the user's new
// password. In one transaction it:
//   - locates the token by hash (ErrNotFound if it never existed or was already used),
//   - rejects an expired token (ErrTokenExpired),
//   - writes the new bcrypt hash to the user,
//   - deletes ALL of that user's reset tokens (single-use + invalidate older links),
//   - deletes ALL of that user's refresh tokens (a password change ends every session).
//
// The whole thing is one txn so a crash can never leave a half-applied reset
// (e.g. password changed but the token still live).
func (s *Store) ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin password reset txn: %w", err)
	}
	defer tx.Rollback()

	var userID int64
	var expiresAt time.Time
	const lookup = `
		SELECT user_id, expires_at FROM password_reset_tokens
		WHERE token_hash = $1 FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, lookup, tokenHash).Scan(&userID, &expiresAt); {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNotFound
	case err != nil:
		return fmt.Errorf("store: looking up reset token: %w", err)
	}
	if time.Now().After(expiresAt) {
		return ErrTokenExpired
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`, newPasswordHash, userID); err != nil {
		return fmt.Errorf("store: updating password: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("store: clearing reset tokens: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("store: revoking sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit password reset: %w", err)
	}
	return nil
}

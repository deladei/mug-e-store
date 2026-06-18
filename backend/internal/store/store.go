// Package store is the persistence layer: it is the only package that writes
// SQL. It exposes typed methods over PostgreSQL and returns plain structs plus
// a small set of sentinel errors that higher layers map onto HTTP responses.
// It depends on internal/domain for the lifecycle rules and nothing above it.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Sentinel errors. Handlers match these with errors.Is to choose a status code
// (e.g. ErrNotFound -> 404, ErrEmailTaken -> 409). Keeping them here means the
// HTTP layer never inspects raw driver errors.
var (
	ErrNotFound           = errors.New("store: not found")
	ErrEmailTaken         = errors.New("store: email already registered")
	ErrDuplicate          = errors.New("store: duplicate")
	ErrDuplicateOrder     = errors.New("store: duplicate order (idempotency key reused)")
	ErrEmptyCart          = errors.New("store: cart is empty")
	ErrUnavailable        = errors.New("store: item is unavailable")
	ErrInvalidTransition  = errors.New("store: invalid status transition")
	ErrInsufficientPoints = errors.New("store: insufficient loyalty points to redeem")
	ErrTokenExpired       = errors.New("store: token has expired")
)

// pgUniqueViolation is the SQLSTATE code Postgres returns on a UNIQUE breach.
const pgUniqueViolation = "23505"

// Store wraps the database handle. All methods take a context so requests can
// carry deadlines and cancellation.
type Store struct {
	db *sql.DB
}

// New wraps an existing *sql.DB (used in tests with a shared handle).
func New(db *sql.DB) *Store { return &Store{db: db} }

// Open connects to Postgres via the lib/pq driver and verifies the connection
// with a Ping, so a bad DATABASE_URL fails at startup, not at first query.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: opening database: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: pinging database: %w", err)
	}
	return &Store{db: db}, nil
}

// DB exposes the underlying handle (for migrations/health checks).
func (s *Store) DB() *sql.DB { return s.db }

// Close releases the connection pool.
func (s *Store) Close() error { return s.db.Close() }

// isUniqueViolation reports whether err is a Postgres UNIQUE violation, and on
// which constraint, so callers can distinguish (e.g.) a taken email from a
// reused idempotency key.
func isUniqueViolation(err error) (constraint string, ok bool) {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && string(pqErr.Code) == pgUniqueViolation {
		return pqErr.Constraint, true
	}
	return "", false
}

// User mirrors a row of the users table. PasswordHash is the bcrypt hash and
// must never be serialized to a client. IsGuest is true for a passwordless
// guest-checkout account (PRD S4): such a user behaves like any customer but can
// never log in and earns no loyalty points.
type User struct {
	ID           int64
	Name         string
	Email        string
	Phone        string
	PasswordHash string
	Role         string
	IsGuest      bool
	CreatedAt    time.Time
}

// CreateUser inserts a user and fills in the generated id and created_at. A
// duplicate email yields ErrEmailTaken. If role is empty, the column default
// ('customer') applies. IsGuest is written as given (false for the normal
// register path; the guest-session handler sets it true).
func (s *Store) CreateUser(ctx context.Context, u *User) error {
	const q = `
		INSERT INTO users (name, email, phone, password_hash, role, is_guest)
		VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5, ''), 'customer'), $6)
		RETURNING id, role, created_at`
	err := s.db.QueryRowContext(ctx, q, u.Name, u.Email, u.Phone, u.PasswordHash, u.Role, u.IsGuest).
		Scan(&u.ID, &u.Role, &u.CreatedAt)
	if c, ok := isUniqueViolation(err); ok && c == "users_email_key" {
		return ErrEmailTaken
	}
	if err != nil {
		return fmt.Errorf("store: creating user: %w", err)
	}
	return nil
}

// GetUserByEmail looks a user up by email for login. A missing user returns
// ErrNotFound; callers must surface the same error as a wrong password so the
// two are indistinguishable (no account enumeration).
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	const q = `
		SELECT id, name, email, phone, password_hash, role, is_guest, created_at
		FROM users WHERE email = $1`
	return s.scanUser(s.db.QueryRowContext(ctx, q, email))
}

// GetUserByID looks a user up by id (e.g. to resolve a token's uid claim).
func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	const q = `
		SELECT id, name, email, phone, password_hash, role, is_guest, created_at
		FROM users WHERE id = $1`
	return s.scanUser(s.db.QueryRowContext(ctx, q, id))
}

func (s *Store) scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.IsGuest, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: scanning user: %w", err)
	}
	return &u, nil
}

// RefreshToken mirrors a row of refresh_tokens. Only the hash is stored; the
// raw token never touches the database.
type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CreateRefreshToken stores a new session's token hash and expiry.
func (s *Store) CreateRefreshToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`
	if _, err := s.db.ExecContext(ctx, q, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("store: creating refresh token: %w", err)
	}
	return nil
}

// GetRefreshToken fetches a session by its token hash. A missing row (already
// rotated, or never existed) returns ErrNotFound.
func (s *Store) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM refresh_tokens WHERE token_hash = $1`
	var rt RefreshToken
	err := s.db.QueryRowContext(ctx, q, tokenHash).
		Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: scanning refresh token: %w", err)
	}
	return &rt, nil
}

// DeleteRefreshToken removes a session by hash. It is the first half of
// rotation (delete the used token, then insert a new one) and makes a refresh
// token single-use. A no-match returns ErrNotFound so a replayed token is
// detectable.
func (s *Store) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	const q = `DELETE FROM refresh_tokens WHERE token_hash = $1`
	res, err := s.db.ExecContext(ctx, q, tokenHash)
	if err != nil {
		return fmt.Errorf("store: deleting refresh token: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Package auth holds the credential primitives: password hashing, access-token
// minting and verification, and refresh-token generation/hashing. It encodes
// the security invariants from CLAUDE.md §4 — bcrypt cost 12, HS256 access
// tokens with the signing method asserted on parse, and refresh tokens that
// are only ever stored as SHA-256 hashes.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	// bcryptCost is fixed at 12 per CLAUDE.md §4.
	bcryptCost = 12
	// accessTokenTTL is the short lifetime of an access token; refresh tokens
	// carry the long-lived session.
	accessTokenTTL = 15 * time.Minute
	// refreshTokenBytes is the entropy of a raw refresh token before encoding.
	refreshTokenBytes = 32
)

// ErrPasswordMismatch is returned by CheckPassword when the password does not
// match the hash. Callers must surface the SAME error for an unknown email so
// the two cases are indistinguishable (no account enumeration).
var ErrPasswordMismatch = errors.New("auth: password does not match")

// HashPassword returns a bcrypt hash (cost 12) of the plaintext password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hashing password: %w", err)
	}
	return string(b), nil
}

// CheckPassword reports whether plain matches the stored bcrypt hash. A
// mismatch yields ErrPasswordMismatch; other (malformed-hash) failures are
// returned as-is.
func CheckPassword(hash, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrPasswordMismatch
	}
	return err
}

// Claims is the access-token payload: the user id and role drive every
// authorization decision downstream.
type Claims struct {
	UID  int64  `json:"uid"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager mints and verifies HS256 access tokens. now is injectable so
// expiry can be tested deterministically.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokenManager builds a TokenManager from the HS256 signing secret, using
// the standard 15-minute access-token TTL and the real clock.
func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    accessTokenTTL,
		now:    time.Now,
	}
}

// GenerateAccessToken issues a signed HS256 token carrying uid and role.
func (m *TokenManager) GenerateAccessToken(uid int64, role string) (string, error) {
	issued := m.now()
	claims := &Claims{
		UID:  uid,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(issued),
			ExpiresAt: jwt.NewNumericDate(issued.Add(m.ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("auth: signing access token: %w", err)
	}
	return signed, nil
}

// ParseAccessToken verifies the signature and expiry and returns the claims.
// It asserts the signing method is HMAC: this defeats the algorithm-confusion
// attack where a token forged with alg=none or an asymmetric alg is presented.
// Expiry is checked against m.now.
func (m *TokenManager) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	keyFunc := func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method %q", t.Header["alg"])
		}
		return m.secret, nil
	}
	_, err := jwt.ParseWithClaims(tokenStr, claims, keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return nil, fmt.Errorf("auth: parsing access token: %w", err)
	}
	return claims, nil
}

// GenerateOpaqueToken returns a fresh, URL-safe random bearer secret. It backs
// both refresh tokens and password-reset tokens: in each case the raw value is
// shown to the client exactly once (a cookie, a reset link) and never stored —
// only HashToken(raw) is persisted.
func GenerateOpaqueToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: generating token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the SHA-256 hex digest of an opaque token. The database
// stores only this; a leak of the table therefore yields no usable secrets. It
// is deterministic so a presented token can be looked up by hash.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateRefreshToken and HashRefreshToken name the session-token use of the
// opaque-token primitives above. They are retained so existing callers read
// intent-first ("this is a refresh token").
func GenerateRefreshToken() (string, error) { return GenerateOpaqueToken() }
func HashRefreshToken(token string) string   { return HashToken(token) }

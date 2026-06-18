package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("hash equals plaintext — not hashed")
	}
	if err := CheckPassword(hash, "correct horse battery staple"); err != nil {
		t.Errorf("CheckPassword() on right password = %v, want nil", err)
	}
	if err := CheckPassword(hash, "wrong password"); !errors.Is(err, ErrPasswordMismatch) {
		t.Errorf("CheckPassword() on wrong password = %v, want ErrPasswordMismatch", err)
	}
}

func TestHashPassword_DistinctSalts(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Error("two hashes of the same password are identical — salt not applied")
	}
}

func newTM() *TokenManager {
	return NewTokenManager("test-signing-secret")
}

func TestAccessToken_RoundTrip(t *testing.T) {
	tm := newTM()
	tok, err := tm.GenerateAccessToken(42, "staff")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}
	claims, err := tm.ParseAccessToken(tok)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.UID != 42 {
		t.Errorf("UID = %d, want 42", claims.UID)
	}
	if claims.Role != "staff" {
		t.Errorf("Role = %q, want staff", claims.Role)
	}
}

func TestAccessToken_Expired(t *testing.T) {
	tm := newTM()
	// Issue a token whose clock is 16 minutes in the past so it has expired
	// against the 15-minute TTL.
	tm.now = func() time.Time { return time.Now().Add(-16 * time.Minute) }
	tok, err := tm.GenerateAccessToken(1, "customer")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}
	tm.now = time.Now // validate against the real clock
	if _, err := tm.ParseAccessToken(tok); err == nil {
		t.Error("ParseAccessToken() on expired token = nil, want error")
	}
}

func TestAccessToken_WrongSecret(t *testing.T) {
	tok, err := NewTokenManager("secret-A").GenerateAccessToken(1, "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}
	if _, err := NewTokenManager("secret-B").ParseAccessToken(tok); err == nil {
		t.Error("ParseAccessToken() with wrong secret = nil, want error")
	}
}

// TestAccessToken_RejectsAlgNone is the critical security test: a token forged
// with alg=none (no signature) must be rejected. The parser asserts the
// signing method is HMAC and never trusts the token's own header.
func TestAccessToken_RejectsAlgNone(t *testing.T) {
	claims := &Claims{
		UID:  1,
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("building alg=none token: %v", err)
	}
	if _, err := newTM().ParseAccessToken(unsigned); err == nil {
		t.Error("ParseAccessToken() accepted an alg=none token — algorithm confusion vulnerability")
	}
}

func TestAccessToken_Garbage(t *testing.T) {
	if _, err := newTM().ParseAccessToken("not.a.jwt"); err == nil {
		t.Error("ParseAccessToken() on garbage = nil, want error")
	}
}

func TestRefreshToken_GenerateAndHash(t *testing.T) {
	t1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}
	t2, _ := GenerateRefreshToken()
	if t1 == t2 {
		t.Error("two refresh tokens are identical — not random")
	}
	if len(t1) < 32 {
		t.Errorf("refresh token length = %d, want >= 32 (sufficient entropy)", len(t1))
	}

	h := HashRefreshToken(t1)
	if h == t1 {
		t.Error("hash equals the token — the raw token must never be what we store")
	}
	if strings.ContainsAny(h, "+/=") {
		t.Errorf("hash %q is not hex", h)
	}
	if HashRefreshToken(t1) != h {
		t.Error("HashRefreshToken is not deterministic — lookup by hash would fail")
	}
	if HashRefreshToken(t2) == h {
		t.Error("different tokens hashed to the same value")
	}
}

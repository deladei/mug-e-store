package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

// sign produces the hex HMAC-SHA512 a genuine Paystack webhook would carry.
func sign(secret string, body []byte) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_Valid(t *testing.T) {
	secret := "sk_test_secret"
	body := []byte(`{"event":"charge.success","data":{"reference":"ref_123"}}`)
	if !VerifySignature(secret, body, sign(secret, body)) {
		t.Error("VerifySignature rejected a genuine signature")
	}
}

func TestVerifySignature_Forged(t *testing.T) {
	secret := "sk_test_secret"
	body := []byte(`{"event":"charge.success"}`)
	// A signature an attacker simply made up.
	forged := hex.EncodeToString([]byte("0000000000000000000000000000000000000000000000000000000000000000"))
	if VerifySignature(secret, body, forged) {
		t.Error("VerifySignature accepted a forged signature")
	}
}

func TestVerifySignature_WrongSecret(t *testing.T) {
	body := []byte(`{"event":"charge.success"}`)
	// Signed with a different secret than the one we verify against.
	sig := sign("attacker_secret", body)
	if VerifySignature("sk_test_real", body, sig) {
		t.Error("VerifySignature accepted a signature made with the wrong secret")
	}
}

func TestVerifySignature_TamperedBody(t *testing.T) {
	secret := "sk_test_secret"
	original := []byte(`{"data":{"amount":1000}}`)
	sig := sign(secret, original)
	tampered := []byte(`{"data":{"amount":999999}}`)
	if VerifySignature(secret, tampered, sig) {
		t.Error("VerifySignature accepted a body that was tampered after signing")
	}
}

func TestVerifySignature_NonHex(t *testing.T) {
	secret := "sk_test_secret"
	body := []byte(`{}`)
	if VerifySignature(secret, body, "not-hex-zzzz") {
		t.Error("VerifySignature accepted a non-hex signature")
	}
}

func TestInitialize_Success(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":true,"message":"ok","data":{"authorization_url":"https://paystack/checkout/abc","access_code":"ac","reference":"ref_xyz"}}`))
	}))
	defer srv.Close()

	c := NewClient("sk_test_key", srv.URL)
	res, err := c.Initialize(context.Background(), InitRequest{
		Email: "ama@example.com", AmountPesewas: 3000, Reference: "ref_xyz",
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if res.AuthorizationURL != "https://paystack/checkout/abc" {
		t.Errorf("AuthorizationURL = %q", res.AuthorizationURL)
	}
	if res.Reference != "ref_xyz" {
		t.Errorf("Reference = %q", res.Reference)
	}
	if gotAuth != "Bearer sk_test_key" {
		t.Errorf("Authorization header = %q, want Bearer sk_test_key", gotAuth)
	}
	if !contains(gotBody, `"amount":3000`) || !contains(gotBody, `"currency":"GHS"`) {
		t.Errorf("request body = %q, want amount in pesewas and GHS currency", gotBody)
	}
}

func TestInitialize_PaystackError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":false,"message":"Invalid key"}`))
	}))
	defer srv.Close()
	c := NewClient("sk_bad", srv.URL)
	if _, err := c.Initialize(context.Background(), InitRequest{Email: "a@b.c", AmountPesewas: 100}); err == nil {
		t.Error("Initialize() error = nil, want error when Paystack returns status=false")
	}
}

func TestVerify_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transaction/verify/ref_xyz" {
			t.Errorf("verify path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"status":true,"data":{"status":"success","amount":3000,"currency":"GHS","reference":"ref_xyz"}}`))
	}))
	defer srv.Close()
	c := NewClient("sk_test_key", srv.URL)
	res, err := c.Verify(context.Background(), "ref_xyz")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if res.Status != "success" || res.AmountPesewas != 3000 || res.Currency != "GHS" {
		t.Errorf("Verify result = %+v", res)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

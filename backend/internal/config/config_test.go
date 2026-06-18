package config

import (
	"strings"
	"testing"
)

// envMap turns a map into the getenv signature Load expects, so tests never
// touch the real process environment.
func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func fullEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":         "postgres://u:p@localhost:5432/coffeemug?sslmode=disable",
		"JWT_SECRET":           "a-long-random-secret",
		"PAYSTACK_SECRET_KEY":  "sk_test_abc123",
		"PAYSTACK_BASE_URL":    "https://api.paystack.co",
		"PORT":                 "9090",
		"DELIVERY_FEE_PESEWAS": "2500",
		"FRONTEND_ORIGIN":      "https://shop.example.com",
	}
}

func TestLoad_FullEnv(t *testing.T) {
	cfg, err := Load(envMap(fullEnv()))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/coffeemug?sslmode=disable" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "a-long-random-secret" {
		t.Errorf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.PaystackSecretKey != "sk_test_abc123" {
		t.Errorf("PaystackSecretKey = %q", cfg.PaystackSecretKey)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.DeliveryFeePesewas != 2500 {
		t.Errorf("DeliveryFeePesewas = %d, want 2500", cfg.DeliveryFeePesewas)
	}
	if cfg.FrontendOrigin != "https://shop.example.com" {
		t.Errorf("FrontendOrigin = %q", cfg.FrontendOrigin)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Only the required vars are set; everything else should default.
	cfg, err := Load(envMap(map[string]string{
		"DATABASE_URL":        "postgres://localhost/db",
		"JWT_SECRET":          "secret",
		"PAYSTACK_SECRET_KEY": "sk_test_x",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.PaystackBaseURL != defaultPaystackBaseURL {
		t.Errorf("PaystackBaseURL = %q, want %q", cfg.PaystackBaseURL, defaultPaystackBaseURL)
	}
	if cfg.Port != defaultPort {
		t.Errorf("Port = %q, want %q", cfg.Port, defaultPort)
	}
	if cfg.DeliveryFeePesewas != defaultDeliveryFeePesewas {
		t.Errorf("DeliveryFeePesewas = %d, want %d", cfg.DeliveryFeePesewas, defaultDeliveryFeePesewas)
	}
	if cfg.FrontendOrigin != defaultFrontendOrigin {
		t.Errorf("FrontendOrigin = %q, want %q", cfg.FrontendOrigin, defaultFrontendOrigin)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// No env at all: the error must name every missing required var so an
	// operator sees all problems at once, not one per restart.
	_, err := Load(envMap(map[string]string{}))
	if err == nil {
		t.Fatal("Load() error = nil, want error for missing required vars")
	}
	for _, want := range []string{"DATABASE_URL", "JWT_SECRET", "PAYSTACK_SECRET_KEY"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention missing %s", err.Error(), want)
		}
	}
}

func TestLoad_MissingOneRequired(t *testing.T) {
	env := fullEnv()
	delete(env, "JWT_SECRET")
	_, err := Load(envMap(env))
	if err == nil {
		t.Fatal("Load() error = nil, want error for missing JWT_SECRET")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("error %q does not mention JWT_SECRET", err.Error())
	}
	if strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q wrongly mentions DATABASE_URL, which was set", err.Error())
	}
}

func TestLoad_BadDeliveryFee(t *testing.T) {
	for _, bad := range []string{"abc", "-100", "12.5"} {
		env := fullEnv()
		env["DELIVERY_FEE_PESEWAS"] = bad
		if _, err := Load(envMap(env)); err == nil {
			t.Errorf("Load() with DELIVERY_FEE_PESEWAS=%q error = nil, want error", bad)
		}
	}
}

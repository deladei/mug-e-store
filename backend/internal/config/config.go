// Package config loads runtime configuration from the environment only
// (12-factor). The service refuses to start if any required secret is absent,
// so a misconfigured deploy fails loudly at boot rather than at first request.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Defaults for the optional variables. The required ones have no default by
// design — their absence is a startup error.
const (
	defaultPaystackBaseURL    = "https://api.paystack.co"
	defaultPort               = "8080"
	defaultDeliveryFeePesewas = int64(1000) // GHS 10.00, per TRD §7
	defaultFrontendOrigin     = "http://localhost:3000"
)

// Config is the fully-resolved runtime configuration.
type Config struct {
	DatabaseURL        string
	JWTSecret          string
	PaystackSecretKey  string
	PaystackBaseURL    string
	Port               string
	DeliveryFeePesewas int64
	FrontendOrigin     string
}

// Load builds a Config from getenv (os.Getenv in production, a stub in tests).
// It returns an error that names every problem at once — all missing required
// variables plus any malformed optional value — so an operator can fix the
// environment in a single pass instead of one restart per mistake.
func Load(getenv func(string) string) (*Config, error) {
	var missing []string
	required := func(key string) string {
		v := getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := &Config{
		DatabaseURL:       required("DATABASE_URL"),
		JWTSecret:         required("JWT_SECRET"),
		PaystackSecretKey: required("PAYSTACK_SECRET_KEY"),
		PaystackBaseURL:   orDefault(getenv("PAYSTACK_BASE_URL"), defaultPaystackBaseURL),
		Port:              orDefault(getenv("PORT"), defaultPort),
		FrontendOrigin:    orDefault(getenv("FRONTEND_ORIGIN"), defaultFrontendOrigin),
	}

	var problems []string
	for _, m := range missing {
		problems = append(problems, fmt.Sprintf("%s is required", m))
	}

	fee, err := parseDeliveryFee(getenv("DELIVERY_FEE_PESEWAS"))
	if err != nil {
		problems = append(problems, err.Error())
	}
	cfg.DeliveryFeePesewas = fee

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid configuration: %s", strings.Join(problems, "; "))
	}
	return cfg, nil
}

// LoadFromEnv is the production entry point: it reads the real process
// environment.
func LoadFromEnv() (*Config, error) { return Load(os.Getenv) }

// parseDeliveryFee resolves DELIVERY_FEE_PESEWAS: empty means the default; a
// set value must be a non-negative integer (money is whole pesewas, never a
// float).
func parseDeliveryFee(raw string) (int64, error) {
	if raw == "" {
		return defaultDeliveryFeePesewas, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return defaultDeliveryFeePesewas, fmt.Errorf("DELIVERY_FEE_PESEWAS must be an integer number of pesewas, got %q", raw)
	}
	if v < 0 {
		return defaultDeliveryFeePesewas, fmt.Errorf("DELIVERY_FEE_PESEWAS must not be negative, got %d", v)
	}
	return v, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// Package paystack is the Paystack integration: a thin client for initializing
// and verifying transactions, plus the HMAC-SHA512 webhook signature check.
// Money crossing this boundary is in pesewas — Paystack's smallest-unit amount
// for GHS is the pesewa, so no conversion is needed.
package paystack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const currencyGHS = "GHS"

// Client talks to the Paystack REST API. baseURL is configurable so tests can
// point it at an httptest server.
type Client struct {
	secretKey string
	baseURL   string
	http      *http.Client
}

// NewClient builds a client for the given secret key and base URL.
func NewClient(secretKey, baseURL string) *Client {
	return &Client{
		secretKey: secretKey,
		baseURL:   baseURL,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

// InitRequest is the input to Initialize. AmountPesewas is the order total.
type InitRequest struct {
	Email         string
	AmountPesewas int64
	Reference     string
	CallbackURL   string
}

// InitResult is what the frontend needs after checkout: where to send the
// browser, and the reference that anchors the webhook.
type InitResult struct {
	AuthorizationURL string
	Reference        string
}

// Initialize creates a Paystack transaction and returns its hosted checkout URL
// and reference. Amount is sent in pesewas with currency GHS.
func (c *Client) Initialize(ctx context.Context, req InitRequest) (*InitResult, error) {
	payload := map[string]any{
		"email":    req.Email,
		"amount":   req.AmountPesewas,
		"currency": currencyGHS,
	}
	if req.Reference != "" {
		payload["reference"] = req.Reference
	}
	if req.CallbackURL != "" {
		payload["callback_url"] = req.CallbackURL
	}

	var out struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			AuthorizationURL string `json:"authorization_url"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, "/transaction/initialize", payload, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack: initialize failed: %s", out.Message)
	}
	return &InitResult{
		AuthorizationURL: out.Data.AuthorizationURL,
		Reference:        out.Data.Reference,
	}, nil
}

// VerifyResult is the server-to-server confirmation of a transaction. The
// caller must still check Status == "success", AmountPesewas == order total,
// and Currency == GHS before marking an order paid.
type VerifyResult struct {
	Status        string
	AmountPesewas int64
	Currency      string
	Reference     string
}

// Verify performs the server-to-server confirmation for a reference. It is the
// second of the four webhook gates — a valid signature is not enough; the truth
// of a payment comes from this call, not from the webhook body.
func (c *Client) Verify(ctx context.Context, reference string) (*VerifyResult, error) {
	var out struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Status    string `json:"status"`
			Amount    int64  `json:"amount"`
			Currency  string `json:"currency"`
			Reference string `json:"reference"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/transaction/verify/"+reference, nil, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack: verify failed: %s", out.Message)
	}
	return &VerifyResult{
		Status:        out.Data.Status,
		AmountPesewas: out.Data.Amount,
		Currency:      out.Data.Currency,
		Reference:     out.Data.Reference,
	}, nil
}

// do issues a request to the Paystack API and decodes the JSON response into
// out. A non-2xx response is an error.
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("paystack: encoding request: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return fmt.Errorf("paystack: building request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("paystack: request to %s: %w", path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("paystack: reading response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("paystack: %s %s returned %d: %s", method, path, resp.StatusCode, raw)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("paystack: decoding response: %w", err)
	}
	return nil
}

// VerifySignature reports whether signature is a genuine Paystack webhook
// signature for body: the hex HMAC-SHA512 of the raw body under the secret key.
// The comparison is constant-time. This is the first of the four webhook gates;
// a request that fails it is discarded before anything else is considered.
func VerifySignature(secretKey string, body []byte, signature string) bool {
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), provided)
}

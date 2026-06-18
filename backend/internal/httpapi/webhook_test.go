package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

const webhookPath = "/api/v1/webhooks/paystack"

// postWebhook sends a charge.success webhook for a reference, signed (or not)
// with the given secret, and returns the response.
func (h *harness) postWebhook(t *testing.T, reference, signingSecret string) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"event": "charge.success",
		"data":  map[string]any{"reference": reference},
	})
	req, err := http.NewRequest(http.MethodPost, h.srv.URL+webhookPath, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Paystack-Signature", signPaystack(signingSecret, body))
	resp, err := h.srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func (h *harness) orderStatus(t *testing.T, id int64) string {
	t.Helper()
	o, err := h.store.GetOrder(context.Background(), id)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	return string(o.Status)
}

// TestWebhook_HappyPath: all four gates pass, so the order flips to paid.
func TestWebhook_HappyPath(t *testing.T) {
	h := newHarness(t)
	u := h.createUser(t, "pay@example.com", "")
	order := h.seedOrder(t, u, "CMUG-PAY-1") // total 2000

	h.verifyStatus = "success"
	h.verifyAmount = order.TotalPesewas
	h.verifyCurrency = "GHS"

	resp := h.postWebhook(t, "CMUG-PAY-1", testPaystackSecret)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	if s := h.orderStatus(t, order.ID); s != "paid" {
		t.Errorf("order status = %s, want paid", s)
	}
}

// TestWebhook_InvalidSignature: gate 1 fails, 401, order untouched.
func TestWebhook_InvalidSignature(t *testing.T) {
	h := newHarness(t)
	u := h.createUser(t, "pay2@example.com", "")
	order := h.seedOrder(t, u, "CMUG-PAY-2")
	h.verifyStatus = "success"
	h.verifyAmount = order.TotalPesewas

	resp := h.postWebhook(t, "CMUG-PAY-2", "the-wrong-secret")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	if s := h.orderStatus(t, order.ID); s != "pending_payment" {
		t.Errorf("order status = %s, want pending_payment (untouched)", s)
	}
}

// TestWebhook_AmountMismatch: gate 3 fails. A valid signature and a successful
// verify, but the verified amount does not equal the order total — the order
// must NOT be marked paid. The handler acknowledges with 200 (a retry cannot
// change the amount) but writes nothing.
func TestWebhook_AmountMismatch(t *testing.T) {
	h := newHarness(t)
	u := h.createUser(t, "pay3@example.com", "")
	order := h.seedOrder(t, u, "CMUG-PAY-3") // total 2000

	h.verifyStatus = "success"
	h.verifyAmount = order.TotalPesewas + 100 // attacker/bug: wrong amount
	h.verifyCurrency = "GHS"

	resp := h.postWebhook(t, "CMUG-PAY-3", testPaystackSecret)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (acknowledged, not retried)", resp.StatusCode)
	}
	resp.Body.Close()

	if s := h.orderStatus(t, order.ID); s != "pending_payment" {
		t.Errorf("order status = %s, want pending_payment (mismatch must not pay)", s)
	}
}

// TestWebhook_VerifyNotSuccessful: gate 2 fails (Paystack says the charge is not
// successful), so the order is not paid.
func TestWebhook_VerifyNotSuccessful(t *testing.T) {
	h := newHarness(t)
	u := h.createUser(t, "pay4@example.com", "")
	order := h.seedOrder(t, u, "CMUG-PAY-4")

	h.verifyStatus = "abandoned" // not "success"
	h.verifyAmount = order.TotalPesewas
	h.verifyCurrency = "GHS"

	resp := h.postWebhook(t, "CMUG-PAY-4", testPaystackSecret)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (ignored)", resp.StatusCode)
	}
	resp.Body.Close()

	if s := h.orderStatus(t, order.ID); s != "pending_payment" {
		t.Errorf("order status = %s, want pending_payment", s)
	}
}

// TestWebhook_Idempotent: a duplicate delivery (Paystack retries) is a no-op
// success — the order stays paid and nothing errors.
func TestWebhook_Idempotent(t *testing.T) {
	h := newHarness(t)
	u := h.createUser(t, "pay5@example.com", "")
	order := h.seedOrder(t, u, "CMUG-PAY-5")
	h.verifyStatus = "success"
	h.verifyAmount = order.TotalPesewas
	h.verifyCurrency = "GHS"

	for i := 0; i < 2; i++ {
		resp := h.postWebhook(t, "CMUG-PAY-5", testPaystackSecret)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("delivery %d status = %d, want 200", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}
	if s := h.orderStatus(t, order.ID); s != "paid" {
		t.Errorf("order status = %s, want paid", s)
	}
}

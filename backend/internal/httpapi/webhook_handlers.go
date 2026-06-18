package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"coffeemug/backend/internal/domain"
	"coffeemug/backend/internal/paystack"
	"coffeemug/backend/internal/sse"
	"coffeemug/backend/internal/store"
)

// currencyGHS is the only currency the shop transacts in; the webhook compares
// the verified currency against it before an order can be marked paid.
const currencyGHS = "GHS"

// paystackEvent is the slice of the Paystack webhook payload we act on. We read
// only the event type and the reference from the body — every figure that
// decides "paid" comes from the server-to-server verify call, never from this
// body, which is attacker-influenced even when a valid signature is replayed.
type paystackEvent struct {
	Event string `json:"event"`
	Data  struct {
		Reference string `json:"reference"`
	} `json:"data"`
}

// handlePaystackWebhook is the only path that can move an order to `paid`. It is
// authenticated by signature, not a token, and enforces the four payment gates
// from TRD §5.2, in order:
//
//  1. a valid x-paystack-signature (HMAC-SHA512 of the raw body under the secret)
//  2. a server-to-server verify call returns status "success"
//  3. the verified amount equals the order total AND the currency is GHS
//  4. the order is in a state from which `paid` is reachable
//
// It is idempotent: Paystack retries deliveries, and re-applying `paid` is a
// no-op (the state machine and the UNIQUE paystack_reference both guarantee an
// order is marked paid at most once). Transient failures return 5xx so Paystack
// retries; permanent rejections return 200 so it stops — a retry can never fix a
// bad signature, an unknown reference, or an amount mismatch.
func (s *Server) handlePaystackWebhook(w http.ResponseWriter, r *http.Request) {
	// The raw bytes must be hashed exactly as received, so read the body before
	// any JSON decoding.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20)) // 1 MiB
	if err != nil {
		writeError(w, http.StatusBadRequest, codeValidation, "could not read request body")
		return
	}

	// Gate 1 — signature. A request that fails it is discarded before the body
	// is even parsed.
	if !paystack.VerifySignature(s.cfg.PaystackSecretKey, body, r.Header.Get("X-Paystack-Signature")) {
		s.logger.Warn("paystack webhook: invalid signature")
		writeError(w, http.StatusUnauthorized, codeUnauthorized, "invalid signature")
		return
	}

	var evt paystackEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		writeError(w, http.StatusBadRequest, codeValidation, "invalid webhook body")
		return
	}
	// We act only on a successful charge. Acknowledge every other event so
	// Paystack stops retrying it.
	if evt.Event != "charge.success" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	if evt.Data.Reference == "" {
		writeError(w, http.StatusBadRequest, codeValidation, "missing transaction reference")
		return
	}

	order, err := s.store.GetOrderByReference(r.Context(), evt.Data.Reference)
	switch {
	case errors.Is(err, store.ErrNotFound):
		// Unknown reference — acknowledge so Paystack stops retrying; a retry
		// will never find an order we never created.
		s.logger.Warn("paystack webhook: unknown reference", "reference", evt.Data.Reference)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	case err != nil:
		s.serverError(w, "loading order for webhook", err)
		return
	}

	// Gate 2 — payment truth comes from the verify call, not the webhook body.
	verified, err := s.paystack.Verify(r.Context(), evt.Data.Reference)
	if err != nil {
		// Transient (network / Paystack outage) — let Paystack retry.
		s.logger.Error("paystack webhook: verify call failed", "reference", evt.Data.Reference, "error", err)
		writeError(w, http.StatusBadGateway, codeInternal, "could not verify transaction")
		return
	}
	if verified.Status != "success" {
		s.logger.Warn("paystack webhook: transaction not successful",
			"reference", evt.Data.Reference, "status", verified.Status)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	// Gate 3 — the money must match exactly: identical integer pesewas, currency
	// GHS. A mismatch is a fraud or bug signal; it is never marked paid, logged
	// loudly, and acknowledged (a retry cannot change the amount).
	if verified.AmountPesewas != order.TotalPesewas || verified.Currency != currencyGHS {
		s.logger.Error("paystack webhook: amount/currency mismatch",
			"reference", evt.Data.Reference,
			"order_id", order.ID,
			"verified_amount", verified.AmountPesewas,
			"order_total", order.TotalPesewas,
			"verified_currency", verified.Currency)
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
		return
	}

	// Gate 4 — the transition. The system is the actor (nil actor_id). paid→paid
	// is a no-op success, which is what makes repeated deliveries safe.
	updated, err := s.store.TransitionOrder(r.Context(), order.ID, domain.StatusPaid, nil)
	switch {
	case errors.Is(err, store.ErrInvalidTransition):
		// The order is in a state from which paid is unreachable (e.g. cancelled).
		// Acknowledge — retrying cannot make the transition legal.
		s.logger.Warn("paystack webhook: order not in a payable state",
			"reference", evt.Data.Reference, "order_id", order.ID, "status", order.Status)
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
		return
	case err != nil:
		s.serverError(w, "transitioning order to paid", err)
		return
	}

	s.broker.Publish(sse.StatusEvent{OrderID: updated.ID, Status: string(updated.Status)})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

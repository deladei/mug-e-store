package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"coffeemug/backend/internal/domain"
	"coffeemug/backend/internal/paystack"
	"coffeemug/backend/internal/store"
)

type checkoutRequest struct {
	Fulfilment     string `json:"fulfilment"`
	Address        string `json:"address"`
	Phone          string `json:"phone"`
	IdempotencyKey string `json:"idempotency_key"`
	// PointsToRedeem is optional; 0 (or omitted) redeems nothing. The discount it
	// produces is decided server-side inside the checkout transaction.
	PointsToRedeem int64 `json:"points_to_redeem"`
}

type checkoutResponse struct {
	Order            *store.Order `json:"order"`
	AuthorizationURL string       `json:"authorization_url"`
}

// handleCheckout turns the cart into a pending order, then initializes payment
// with Paystack. If initialization fails the order is cancelled so it cannot
// dangle, and the client gets 502 payment_init_failed.
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	var req checkoutRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	fulfilment := domain.Fulfilment(req.Fulfilment)
	if !domain.ValidFulfilment(fulfilment) {
		writeError(w, http.StatusBadRequest, codeValidation, "fulfilment must be 'pickup' or 'delivery'")
		return
	}
	req.Address = strings.TrimSpace(req.Address)
	req.Phone = strings.TrimSpace(req.Phone)
	if fulfilment == domain.FulfilmentDelivery && (req.Address == "" || req.Phone == "") {
		writeError(w, http.StatusBadRequest, codeValidation, "delivery requires an address and phone number")
		return
	}
	if req.PointsToRedeem < 0 {
		writeError(w, http.StatusBadRequest, codeValidation, "points_to_redeem cannot be negative")
		return
	}

	order, err := s.store.Checkout(r.Context(), store.CheckoutParams{
		UserID:             u.ID,
		Fulfilment:         fulfilment,
		Address:            req.Address,
		Phone:              req.Phone,
		DeliveryFeePesewas: s.cfg.DeliveryFeePesewas,
		IdempotencyKey:     req.IdempotencyKey,
		RedeemPoints:       req.PointsToRedeem,
	})
	switch {
	case errors.Is(err, store.ErrEmptyCart):
		writeError(w, http.StatusBadRequest, codeEmptyCart, "your cart is empty")
		return
	case errors.Is(err, store.ErrUnavailable):
		writeError(w, http.StatusConflict, codeUnavailable, "an item in your cart is no longer available")
		return
	case errors.Is(err, store.ErrInsufficientPoints):
		writeError(w, http.StatusConflict, codeInsufficientPoints, "you do not have enough loyalty points to redeem that many")
		return
	case errors.Is(err, store.ErrDuplicateOrder):
		writeError(w, http.StatusConflict, codeDuplicateOrder, "this checkout was already submitted")
		return
	case err != nil:
		s.serverError(w, "checkout", err)
		return
	}

	// Initialize payment. On any failure, cancel the order so it cannot dangle.
	user, err := s.store.GetUserByID(r.Context(), u.ID)
	if err != nil {
		s.cancelDangling(r, order.ID)
		s.serverError(w, "loading user for checkout", err)
		return
	}
	reference := newPaystackReference(order.ID)
	callback := strings.TrimRight(s.cfg.FrontendOrigin, "/") + fmt.Sprintf("/orders/%d", order.ID)
	init, err := s.paystack.Initialize(r.Context(), paystack.InitRequest{
		Email:         user.Email,
		AmountPesewas: order.TotalPesewas,
		Reference:     reference,
		CallbackURL:   callback,
	})
	if err != nil {
		s.cancelDangling(r, order.ID)
		s.logger.Error("paystack initialize failed", "order_id", order.ID, "error", err)
		writeError(w, http.StatusBadGateway, codePaymentInitFailed, "could not start payment, please try again")
		return
	}

	if err := s.store.SetPaystackReference(r.Context(), order.ID, init.Reference); err != nil {
		s.serverError(w, "saving paystack reference", err)
		return
	}
	order.PaystackReference = &init.Reference
	writeJSON(w, http.StatusCreated, checkoutResponse{Order: order, AuthorizationURL: init.AuthorizationURL})
}

// cancelDangling cancels an order whose payment could not be initialized. The
// transition is a system action (nil actor).
func (s *Server) cancelDangling(r *http.Request, orderID int64) {
	if _, err := s.store.TransitionOrder(r.Context(), orderID, domain.StatusCancelled, nil); err != nil {
		s.logger.Error("cancelling dangling order", "order_id", orderID, "error", err)
	}
}

// newPaystackReference builds a unique, human-traceable reference for an order.
func newPaystackReference(orderID int64) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("CMUG-%d-%s", orderID, hex.EncodeToString(b))
}

package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"coffeemug/backend/internal/domain"
	"coffeemug/backend/internal/sse"
	"coffeemug/backend/internal/store"
)

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	order, err := s.store.GetOrderForUser(r.Context(), id, u.ID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "order not found")
	case err != nil:
		s.serverError(w, "getting order", err)
	default:
		writeJSON(w, http.StatusOK, order)
	}
}

func (s *Server) handleListMyOrders(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			page = p
		}
	}
	orders, err := s.store.ListUserOrders(r.Context(), u.ID, page)
	if err != nil {
		s.serverError(w, "listing my orders", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders, "page": page})
}

func (s *Server) handleLoyalty(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	balance, err := s.store.LoyaltyBalance(r.Context(), u.ID)
	if err != nil {
		s.serverError(w, "loyalty balance", err)
		return
	}
	ledger, err := s.store.LoyaltyLedger(r.Context(), u.ID)
	if err != nil {
		s.serverError(w, "loyalty ledger", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"balance": balance, "ledger": ledger})
}

// handleOrderEvents streams an order's status as Server-Sent Events. It
// authorizes ownership (a foreign id is 404, not 403), sends an immediate
// snapshot so a late client is never blank, then pushes each transition with a
// heartbeat every 25s to keep proxies from closing the stream. The subscription
// is cleaned up on disconnect (no leak).
func (s *Server) handleOrderEvents(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	order, err := s.store.GetOrderForUser(r.Context(), id, u.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, codeNotFound, "order not found")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, codeInternal, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Subscribe BEFORE sending the snapshot so no transition is missed in the
	// gap between reading current status and starting to listen.
	events, unsubscribe := s.broker.Subscribe(id)
	defer unsubscribe()

	writeSSE(w, sse.StatusEvent{OrderID: order.ID, Status: string(order.Status)})
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-events:
			writeSSE(w, ev)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// writeSSE writes one `event: status` frame.
func writeSSE(w http.ResponseWriter, ev sse.StatusEvent) {
	fmt.Fprintf(w, "event: status\ndata: {\"order_id\":%d,\"status\":%q}\n\n", ev.OrderID, ev.Status)
}

// --- Staff/admin ---

func (s *Server) handleAdminListOrders(w http.ResponseWriter, r *http.Request) {
	var status *domain.Status
	if raw := r.URL.Query().Get("status"); raw != "" {
		st := domain.Status(raw)
		if !domain.ValidStatus(st) {
			writeError(w, http.StatusBadRequest, codeValidation, "unknown status")
			return
		}
		status = &st
	}
	orders, err := s.store.ListOrders(r.Context(), status)
	if err != nil {
		s.serverError(w, "listing orders", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (s *Server) handleAdminOrderHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	events, err := s.store.GetOrderEvents(r.Context(), id)
	if err != nil {
		s.serverError(w, "order history", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": events})
}

type transitionRequest struct {
	To string `json:"to"`
}

func (s *Server) handleAdminTransition(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	var req transitionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	to := domain.Status(req.To)
	if !domain.ValidStatus(to) {
		writeError(w, http.StatusBadRequest, codeValidation, "unknown target status")
		return
	}
	// `paid` is reachable ONLY via the payment webhook. No staff member can set
	// it by hand, regardless of the state machine.
	if to == domain.StatusPaid {
		writeError(w, http.StatusForbidden, codeForbidden, "an order can only be marked paid by the payment webhook")
		return
	}

	actorID := u.ID
	order, err := s.store.TransitionOrder(r.Context(), id, to, &actorID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "order not found")
		return
	case errors.Is(err, store.ErrInvalidTransition):
		writeError(w, http.StatusConflict, codeInvalidTransition, "that status change is not allowed")
		return
	case err != nil:
		s.serverError(w, "transitioning order", err)
		return
	}

	s.broker.Publish(sse.StatusEvent{OrderID: order.ID, Status: string(order.Status)})
	writeJSON(w, http.StatusOK, order)
}

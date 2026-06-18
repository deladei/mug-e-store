package httpapi

import (
	"errors"
	"net/http"

	"coffeemug/backend/internal/store"
)

func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	cart, err := s.store.GetCart(r.Context(), u.ID)
	if err != nil {
		s.serverError(w, "getting cart", err)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

type addCartItemRequest struct {
	ItemVariantID int64 `json:"item_variant_id"`
	Quantity      int   `json:"quantity"`
}

func (s *Server) handleAddCartItem(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	var req addCartItemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ItemVariantID == 0 || req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, codeValidation, "item_variant_id and a positive quantity are required")
		return
	}
	cart, err := s.store.AddCartItem(r.Context(), u.ID, req.ItemVariantID, req.Quantity)
	switch {
	case errors.Is(err, store.ErrUnavailable):
		writeError(w, http.StatusConflict, codeUnavailable, "that item is unavailable")
	case err != nil:
		s.serverError(w, "adding cart item", err)
	default:
		writeJSON(w, http.StatusOK, cart)
	}
}

type updateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

func (s *Server) handleUpdateCartItem(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	lineID, ok := s.pathInt(w, r, "lineId")
	if !ok {
		return
	}
	var req updateCartItemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, codeValidation, "quantity must be positive")
		return
	}
	cart, err := s.store.UpdateCartLine(r.Context(), u.ID, lineID, req.Quantity)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "cart line not found")
	case err != nil:
		s.serverError(w, "updating cart line", err)
	default:
		writeJSON(w, http.StatusOK, cart)
	}
}

func (s *Server) handleRemoveCartItem(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	lineID, ok := s.pathInt(w, r, "lineId")
	if !ok {
		return
	}
	cart, err := s.store.RemoveCartLine(r.Context(), u.ID, lineID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "cart line not found")
	case err != nil:
		s.serverError(w, "removing cart line", err)
	default:
		writeJSON(w, http.StatusOK, cart)
	}
}

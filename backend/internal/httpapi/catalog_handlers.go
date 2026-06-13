package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"coffeemug/backend/internal/store"
)

// showAllItems reports whether the caller may see unavailable items. Catalog
// reads are public, but a valid staff/admin token reveals sold-out items too;
// a customer or anonymous caller sees only available ones.
func (s *Server) showAllItems(r *http.Request) bool {
	token := bearerToken(r)
	if token == "" {
		return false
	}
	claims, err := s.tokens.ParseAccessToken(token)
	if err != nil {
		return false
	}
	return claims.Role == "staff" || claims.Role == "admin"
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.store.ListCategories(r.Context())
	if err != nil {
		s.serverError(w, "listing categories", err)
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	var categoryID *int64
	if raw := r.URL.Query().Get("category"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, codeValidation, "category must be an integer id")
			return
		}
		categoryID = &id
	}
	items, err := s.store.ListItems(r.Context(), categoryID, !s.showAllItems(r))
	if err != nil {
		s.serverError(w, "listing items", err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.store.GetItem(r.Context(), id, !s.showAllItems(r))
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "item not found")
	case err != nil:
		s.serverError(w, "getting item", err)
	default:
		writeJSON(w, http.StatusOK, item)
	}
}

// --- Admin: categories ---

type categoryRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, codeValidation, "name is required")
		return
	}
	c := &store.Category{Name: req.Name, SortOrder: req.SortOrder}
	switch err := s.store.CreateCategory(r.Context(), c); {
	case errors.Is(err, store.ErrDuplicate):
		writeError(w, http.StatusConflict, codeDuplicate, "a category with that name already exists")
	case err != nil:
		s.serverError(w, "creating category", err)
	default:
		writeJSON(w, http.StatusCreated, c)
	}
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	var req categoryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, codeValidation, "name is required")
		return
	}
	c := &store.Category{ID: id, Name: req.Name, SortOrder: req.SortOrder}
	switch err := s.store.UpdateCategory(r.Context(), c); {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "category not found")
	case errors.Is(err, store.ErrDuplicate):
		writeError(w, http.StatusConflict, codeDuplicate, "a category with that name already exists")
	case err != nil:
		s.serverError(w, "updating category", err)
	default:
		writeJSON(w, http.StatusOK, c)
	}
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	switch err := s.store.DeleteCategory(r.Context(), id); {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "category not found")
	case err != nil:
		s.serverError(w, "deleting category", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Admin: items ---

type itemRequest struct {
	CategoryID  int64  `json:"category_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req itemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || req.CategoryID == 0 {
		writeError(w, http.StatusBadRequest, codeValidation, "name and category_id are required")
		return
	}
	it := &store.Item{CategoryID: req.CategoryID, Name: req.Name, Description: req.Description, ImageURL: req.ImageURL}
	if err := s.store.CreateItem(r.Context(), it); err != nil {
		s.serverError(w, "creating item", err)
		return
	}
	writeJSON(w, http.StatusCreated, it)
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	var req itemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || req.CategoryID == 0 {
		writeError(w, http.StatusBadRequest, codeValidation, "name and category_id are required")
		return
	}
	it := &store.Item{ID: id, CategoryID: req.CategoryID, Name: req.Name, Description: req.Description, ImageURL: req.ImageURL}
	switch err := s.store.UpdateItem(r.Context(), it); {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "item not found")
	case err != nil:
		s.serverError(w, "updating item", err)
	default:
		updated, err := s.store.GetItem(r.Context(), id, false)
		if err != nil {
			s.serverError(w, "reloading item", err)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	switch err := s.store.DeleteItem(r.Context(), id); {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "item not found")
	case err != nil:
		s.serverError(w, "deleting item", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

type availabilityRequest struct {
	IsAvailable bool `json:"is_available"`
}

func (s *Server) handleSetAvailability(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	var req availabilityRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.store.SetItemAvailability(r.Context(), id, req.IsAvailable)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "item not found")
	case err != nil:
		s.serverError(w, "setting availability", err)
	default:
		writeJSON(w, http.StatusOK, item)
	}
}

// --- Admin: variants ---

type variantRequest struct {
	Name         string `json:"name"`
	PricePesewas int64  `json:"price_pesewas"`
	SortOrder    int    `json:"sort_order"`
}

func (s *Server) handleCreateVariant(w http.ResponseWriter, r *http.Request) {
	itemID, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	var req variantRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, codeValidation, "name is required")
		return
	}
	if req.PricePesewas < 0 {
		writeError(w, http.StatusBadRequest, codeValidation, "price must not be negative")
		return
	}
	v := &store.Variant{ItemID: itemID, Name: req.Name, PricePesewas: req.PricePesewas, SortOrder: req.SortOrder}
	if err := s.store.CreateVariant(r.Context(), v); err != nil {
		s.serverError(w, "creating variant", err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleDeleteVariant(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathInt(w, r, "id")
	if !ok {
		return
	}
	switch err := s.store.DeleteVariant(r.Context(), id); {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "variant not found")
	case err != nil:
		s.serverError(w, "deleting variant", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

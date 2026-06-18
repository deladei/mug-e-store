package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// CartLine is one priced row of a cart view. Price and names are resolved from
// the live menu at read time — the cart itself stores no price. `Available`
// reflects the item's current sold-out toggle so the UI can flag a line that
// went off-menu after it was added.
type CartLine struct {
	LineID           int64  `json:"line_id"`
	ItemVariantID    int64  `json:"item_variant_id"`
	ItemName         string `json:"item_name"`
	VariantName      string `json:"variant_name"`
	UnitPricePesewas int64  `json:"unit_price_pesewas"`
	Quantity         int    `json:"quantity"`
	Available        bool   `json:"available"`
}

// CartView is the full cart returned to the client after any mutation, so the
// frontend can replace its state from the response without re-fetching.
type CartView struct {
	Lines           []CartLine `json:"lines"`
	SubtotalPesewas int64      `json:"subtotal_pesewas"`
}

// getOrCreateCartID returns the user's cart id, creating the (one-per-user)
// cart on first use. ON CONFLICT keeps it race-safe under the UNIQUE(user_id).
func (s *Store) getOrCreateCartID(ctx context.Context, q queryer, userID int64) (int64, error) {
	const ins = `
		INSERT INTO carts (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id`
	var id int64
	if err := q.QueryRowContext(ctx, ins, userID).Scan(&id); err != nil {
		return 0, fmt.Errorf("store: getting cart: %w", err)
	}
	return id, nil
}

// GetCart returns the user's current cart view (creating an empty cart if none
// exists). Lines are priced and named from the live menu.
func (s *Store) GetCart(ctx context.Context, userID int64) (*CartView, error) {
	cartID, err := s.getOrCreateCartID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	return s.cartView(ctx, s.db, cartID)
}

// cartView builds the priced cart view for a cart id using the given queryer
// (so it works both standalone and inside the checkout transaction).
func (s *Store) cartView(ctx context.Context, q queryer, cartID int64) (*CartView, error) {
	const sel = `
		SELECT cl.id, cl.item_variant_id, i.name, iv.name, iv.price_pesewas,
		       cl.quantity, i.is_available
		FROM cart_lines cl
		JOIN item_variants iv ON iv.id = cl.item_variant_id
		JOIN items i ON i.id = iv.item_id
		WHERE cl.cart_id = $1
		ORDER BY cl.id`
	rows, err := q.QueryContext(ctx, sel, cartID)
	if err != nil {
		return nil, fmt.Errorf("store: reading cart: %w", err)
	}
	defer rows.Close()

	view := &CartView{Lines: []CartLine{}}
	for rows.Next() {
		var l CartLine
		if err := rows.Scan(&l.LineID, &l.ItemVariantID, &l.ItemName, &l.VariantName,
			&l.UnitPricePesewas, &l.Quantity, &l.Available); err != nil {
			return nil, fmt.Errorf("store: scanning cart line: %w", err)
		}
		view.Lines = append(view.Lines, l)
		view.SubtotalPesewas += l.UnitPricePesewas * int64(l.Quantity)
	}
	return view, rows.Err()
}

// AddCartItem adds quantity of a variant to the user's cart and returns the
// updated cart. Adding the same variant again increments the existing line
// (UNIQUE(cart_id, item_variant_id)). An unavailable or unknown variant is
// rejected with ErrUnavailable — availability is enforced server-side, not
// just in the UI.
func (s *Store) AddCartItem(ctx context.Context, userID, variantID int64, quantity int) (*CartView, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("store: quantity must be positive")
	}
	cartID, err := s.getOrCreateCartID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}

	// Reject a variant whose item is unavailable (or which does not exist).
	var available bool
	const chk = `
		SELECT i.is_available
		FROM item_variants iv JOIN items i ON i.id = iv.item_id
		WHERE iv.id = $1`
	switch err := s.db.QueryRowContext(ctx, chk, variantID).Scan(&available); {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrUnavailable
	case err != nil:
		return nil, fmt.Errorf("store: checking variant: %w", err)
	case !available:
		return nil, ErrUnavailable
	}

	const ins = `
		INSERT INTO cart_lines (cart_id, item_variant_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (cart_id, item_variant_id)
		DO UPDATE SET quantity = cart_lines.quantity + EXCLUDED.quantity`
	if _, err := s.db.ExecContext(ctx, ins, cartID, variantID, quantity); err != nil {
		return nil, fmt.Errorf("store: adding cart item: %w", err)
	}
	return s.cartView(ctx, s.db, cartID)
}

// UpdateCartLine sets the quantity of one line the user owns and returns the
// updated cart. A line not in the user's cart yields ErrNotFound.
func (s *Store) UpdateCartLine(ctx context.Context, userID, lineID int64, quantity int) (*CartView, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("store: quantity must be positive")
	}
	cartID, err := s.getOrCreateCartID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	const q = `UPDATE cart_lines SET quantity = $3 WHERE id = $1 AND cart_id = $2`
	res, err := s.db.ExecContext(ctx, q, lineID, cartID, quantity)
	if err != nil {
		return nil, fmt.Errorf("store: updating cart line: %w", err)
	}
	if err := errIfNoRows(res); err != nil {
		return nil, err
	}
	return s.cartView(ctx, s.db, cartID)
}

// RemoveCartLine deletes one line the user owns and returns the updated cart.
func (s *Store) RemoveCartLine(ctx context.Context, userID, lineID int64) (*CartView, error) {
	cartID, err := s.getOrCreateCartID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	const q = `DELETE FROM cart_lines WHERE id = $1 AND cart_id = $2`
	res, err := s.db.ExecContext(ctx, q, lineID, cartID)
	if err != nil {
		return nil, fmt.Errorf("store: removing cart line: %w", err)
	}
	if err := errIfNoRows(res); err != nil {
		return nil, err
	}
	return s.cartView(ctx, s.db, cartID)
}

// queryer is the subset of *sql.DB / *sql.Tx used by helpers that must run
// either standalone or inside a transaction.
type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// Category mirrors a row of categories.
type Category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// Variant is a purchasable size/option of an item; the price lives here.
type Variant struct {
	ID           int64  `json:"id"`
	ItemID       int64  `json:"-"`
	Name         string `json:"name"`
	PricePesewas int64  `json:"price_pesewas"`
	SortOrder    int    `json:"sort_order"`
}

// Item mirrors a row of items together with its variants.
type Item struct {
	ID          int64     `json:"id"`
	CategoryID  int64     `json:"category_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url"`
	IsAvailable bool      `json:"is_available"`
	Variants    []Variant `json:"variants"`
}

// ListCategories returns all categories in display order.
func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	const q = `SELECT id, name, sort_order FROM categories ORDER BY sort_order, name`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: listing categories: %w", err)
	}
	defer rows.Close()

	cats := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("store: scanning category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// CreateCategory inserts a category; a duplicate name yields ErrDuplicate.
func (s *Store) CreateCategory(ctx context.Context, c *Category) error {
	const q = `INSERT INTO categories (name, sort_order) VALUES ($1, $2) RETURNING id`
	err := s.db.QueryRowContext(ctx, q, c.Name, c.SortOrder).Scan(&c.ID)
	if _, ok := isUniqueViolation(err); ok {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("store: creating category: %w", err)
	}
	return nil
}

// UpdateCategory renames/re-sorts a category. A missing id yields ErrNotFound,
// a name clash ErrDuplicate.
func (s *Store) UpdateCategory(ctx context.Context, c *Category) error {
	const q = `UPDATE categories SET name = $2, sort_order = $3 WHERE id = $1`
	res, err := s.db.ExecContext(ctx, q, c.ID, c.Name, c.SortOrder)
	if _, ok := isUniqueViolation(err); ok {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("store: updating category: %w", err)
	}
	return errIfNoRows(res)
}

// DeleteCategory removes a category. A missing id yields ErrNotFound.
func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: deleting category: %w", err)
	}
	return errIfNoRows(res)
}

// ListItems returns items with their variants. When categoryID is non-nil it
// filters to that category. When onlyAvailable is true (the customer path) it
// returns only available items — an unavailable item is absent, not flagged.
func (s *Store) ListItems(ctx context.Context, categoryID *int64, onlyAvailable bool) ([]Item, error) {
	q := `SELECT id, category_id, name, description, image_url, is_available
	      FROM items WHERE 1=1`
	var args []any
	if categoryID != nil {
		args = append(args, *categoryID)
		q += fmt.Sprintf(" AND category_id = $%d", len(args))
	}
	if onlyAvailable {
		q += " AND is_available = TRUE"
	}
	q += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: listing items: %w", err)
	}
	defer rows.Close()

	items := []Item{}
	byID := map[int64]*Item{}
	var ids []int64
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.CategoryID, &it.Name, &it.Description, &it.ImageURL, &it.IsAvailable); err != nil {
			return nil, fmt.Errorf("store: scanning item: %w", err)
		}
		it.Variants = []Variant{}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		byID[items[i].ID] = &items[i]
		ids = append(ids, items[i].ID)
	}
	if err := s.attachVariants(ctx, byID, ids); err != nil {
		return nil, err
	}
	return items, nil
}

// GetItem returns a single item with its variants. When onlyAvailable is true,
// an unavailable item is reported as ErrNotFound (the customer cannot see it).
func (s *Store) GetItem(ctx context.Context, id int64, onlyAvailable bool) (*Item, error) {
	q := `SELECT id, category_id, name, description, image_url, is_available
	      FROM items WHERE id = $1`
	if onlyAvailable {
		q += " AND is_available = TRUE"
	}
	var it Item
	err := s.db.QueryRowContext(ctx, q, id).
		Scan(&it.ID, &it.CategoryID, &it.Name, &it.Description, &it.ImageURL, &it.IsAvailable)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: scanning item: %w", err)
	}
	it.Variants = []Variant{}
	if err := s.attachVariants(ctx, map[int64]*Item{it.ID: &it}, []int64{it.ID}); err != nil {
		return nil, err
	}
	return &it, nil
}

// attachVariants loads variants for the given item ids and appends them to the
// matching items, ordered for display. It runs one query regardless of item
// count (avoids the N+1 pattern).
func (s *Store) attachVariants(ctx context.Context, byID map[int64]*Item, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	const q = `
		SELECT id, item_id, name, price_pesewas, sort_order
		FROM item_variants WHERE item_id = ANY($1)
		ORDER BY sort_order, name`
	rows, err := s.db.QueryContext(ctx, q, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("store: listing variants: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v Variant
		if err := rows.Scan(&v.ID, &v.ItemID, &v.Name, &v.PricePesewas, &v.SortOrder); err != nil {
			return fmt.Errorf("store: scanning variant: %w", err)
		}
		if it := byID[v.ItemID]; it != nil {
			it.Variants = append(it.Variants, v)
		}
	}
	return rows.Err()
}

// CreateItem inserts a menu item (variants are added separately).
func (s *Store) CreateItem(ctx context.Context, it *Item) error {
	const q = `
		INSERT INTO items (category_id, name, description, image_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, is_available`
	err := s.db.QueryRowContext(ctx, q, it.CategoryID, it.Name, it.Description, it.ImageURL).
		Scan(&it.ID, &it.IsAvailable)
	if err != nil {
		return fmt.Errorf("store: creating item: %w", err)
	}
	it.Variants = []Variant{}
	return nil
}

// UpdateItem edits an item's catalog fields. A missing id yields ErrNotFound.
func (s *Store) UpdateItem(ctx context.Context, it *Item) error {
	const q = `
		UPDATE items SET category_id = $2, name = $3, description = $4, image_url = $5
		WHERE id = $1`
	res, err := s.db.ExecContext(ctx, q, it.ID, it.CategoryID, it.Name, it.Description, it.ImageURL)
	if err != nil {
		return fmt.Errorf("store: updating item: %w", err)
	}
	return errIfNoRows(res)
}

// SetItemAvailability flips the sold-out toggle and returns the updated item.
func (s *Store) SetItemAvailability(ctx context.Context, id int64, available bool) (*Item, error) {
	const q = `UPDATE items SET is_available = $2 WHERE id = $1`
	res, err := s.db.ExecContext(ctx, q, id, available)
	if err != nil {
		return nil, fmt.Errorf("store: setting availability: %w", err)
	}
	if err := errIfNoRows(res); err != nil {
		return nil, err
	}
	return s.GetItem(ctx, id, false)
}

// DeleteItem removes an item (its variants cascade). Missing id -> ErrNotFound.
func (s *Store) DeleteItem(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: deleting item: %w", err)
	}
	return errIfNoRows(res)
}

// CreateVariant adds a priced variant to an item.
func (s *Store) CreateVariant(ctx context.Context, v *Variant) error {
	const q = `
		INSERT INTO item_variants (item_id, name, price_pesewas, sort_order)
		VALUES ($1, $2, $3, $4) RETURNING id`
	err := s.db.QueryRowContext(ctx, q, v.ItemID, v.Name, v.PricePesewas, v.SortOrder).Scan(&v.ID)
	if err != nil {
		return fmt.Errorf("store: creating variant: %w", err)
	}
	return nil
}

// DeleteVariant removes a variant. Missing id -> ErrNotFound.
func (s *Store) DeleteVariant(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM item_variants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: deleting variant: %w", err)
	}
	return errIfNoRows(res)
}

// errIfNoRows turns a zero-rows-affected result into ErrNotFound.
func errIfNoRows(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Command seed populates a fresh database with the demonstration data the PRD's
// demo loop and the implementation plan (§2) assume: a realistic Ghanaian
// coffee-shop menu (8 items / 14 variants, all priced in integer pesewas) plus
// the two staff accounts the frontend and the viva walk-through log in as.
//
// It owns only composition: it needs nothing but a database (DATABASE_URL), not
// the JWT or Paystack secrets the API needs, so it can run standalone right
// after `make migrate-up`. All writes go through internal/store — no SQL lives
// here — keeping the strict layering of CLAUDE.md §4 intact.
//
// It is idempotent: the menu is seeded only when the catalog is empty and each
// staff account only when it does not already exist, so re-running it on an
// already-seeded database is a safe no-op rather than a pile of duplicates.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"coffeemug/backend/internal/auth"
	"coffeemug/backend/internal/store"
)

// seedPassword is the shared demo password for the seeded staff accounts. It is
// intentionally well-known and for local/demo use only — production accounts are
// created through the real register/admin flows, never this seeder.
const seedPassword = "password123"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set (copy .env.example to .env and fill it in)")
	}

	st, err := store.Open(dsn)
	if err != nil {
		return err
	}
	defer st.Close()

	ctx := context.Background()
	if err := st.DB().PingContext(ctx); err != nil {
		return fmt.Errorf("database unreachable (did you run make migrate-up?): %w", err)
	}

	if err := seedStaff(ctx, logger, st); err != nil {
		return err
	}
	if err := seedMenu(ctx, logger, st); err != nil {
		return err
	}
	logger.Info("seed complete")
	return nil
}

// seedStaff creates the demo admin and staff accounts if they are absent.
func seedStaff(ctx context.Context, logger *slog.Logger, st *store.Store) error {
	hash, err := auth.HashPassword(seedPassword)
	if err != nil {
		return fmt.Errorf("hashing seed password: %w", err)
	}
	staff := []struct{ name, email, phone, role string }{
		{"Demo Admin", "admin@coffeemug.shop", "0200000001", "admin"},
		{"Demo Staff", "staff@coffeemug.shop", "0200000002", "staff"},
	}
	for _, s := range staff {
		if _, err := st.GetUserByEmail(ctx, s.email); err == nil {
			logger.Info("staff already present, skipping", "email", s.email)
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("checking %s: %w", s.email, err)
		}
		u := &store.User{Name: s.name, Email: s.email, Phone: s.phone, PasswordHash: hash, Role: s.role}
		if err := st.CreateUser(ctx, u); err != nil {
			return fmt.Errorf("creating %s: %w", s.email, err)
		}
		logger.Info("created staff", "email", s.email, "role", s.role)
	}
	return nil
}

// menuItem is one item plus its variants, expressed in the seed's own terms so
// the data reads as a menu. Prices are integer pesewas (1 GHS = 100 pesewas).
type menuItem struct {
	name        string
	description string
	variants    []store.Variant
}

// menuCategory groups items for the storefront, in display order.
type menuCategory struct {
	name  string
	items []menuItem
}

// v is a small constructor so the menu table below stays readable.
func v(name string, pricePesewas int64, sort int) store.Variant {
	return store.Variant{Name: name, PricePesewas: pricePesewas, SortOrder: sort}
}

// demoMenu is the eight-item Ghanaian coffee-shop menu (14 variants) the schema
// doc calls for. The shop is in Accra, so the menu leans local: sobolo, cocoa,
// bofrot. Prices are realistic Accra café prices in pesewas.
var demoMenu = []menuCategory{
	{name: "Coffee", items: []menuItem{
		{"Accra Espresso", "House espresso from West African beans.", []store.Variant{
			v("Single", 1200, 1), v("Double", 1800, 2),
		}},
		{"Cappuccino", "Espresso with steamed milk and a thick foam cap.", []store.Variant{
			v("Regular", 2000, 1), v("Large", 2500, 2),
		}},
		{"Café Latte", "Smooth espresso with plenty of steamed milk.", []store.Variant{
			v("Regular", 2200, 1), v("Large", 2700, 2),
		}},
		{"Ghana Cocoa Mocha", "Espresso and steamed milk with local cocoa.", []store.Variant{
			v("Regular", 2500, 1), v("Large", 3000, 2),
		}},
	}},
	{name: "Tea & Infusions", items: []menuItem{
		{"Lemongrass Tea", "Fresh lemongrass infusion, lightly sweetened.", []store.Variant{
			v("Cup", 1400, 1),
		}},
	}},
	{name: "Cold Drinks", items: []menuItem{
		{"Hibiscus Sobolo", "Chilled hibiscus drink with ginger and cloves.", []store.Variant{
			v("Regular", 1500, 1), v("Large", 2000, 2),
		}},
	}},
	{name: "Pastries", items: []menuItem{
		{"Bofrot", "Ghanaian fried doughnut, soft and lightly spiced.", []store.Variant{
			v("Half dozen", 1000, 1), v("Dozen", 1800, 2),
		}},
		{"Chocolate Croissant", "Buttery croissant with a dark chocolate centre.", []store.Variant{
			v("Each", 1600, 1),
		}},
	}},
}

// seedMenu writes the demo catalog, but only when no categories exist yet, so a
// re-run never duplicates the menu.
func seedMenu(ctx context.Context, logger *slog.Logger, st *store.Store) error {
	existing, err := st.ListCategories(ctx)
	if err != nil {
		return fmt.Errorf("listing categories: %w", err)
	}
	if len(existing) > 0 {
		logger.Info("catalog already seeded, skipping menu", "categories", len(existing))
		return nil
	}

	var items, variants int
	for ci, cat := range demoMenu {
		c := &store.Category{Name: cat.name, SortOrder: ci + 1}
		if err := st.CreateCategory(ctx, c); err != nil {
			return fmt.Errorf("creating category %q: %w", cat.name, err)
		}
		for _, mi := range cat.items {
			it := &store.Item{CategoryID: c.ID, Name: mi.name, Description: mi.description}
			if err := st.CreateItem(ctx, it); err != nil {
				return fmt.Errorf("creating item %q: %w", mi.name, err)
			}
			items++
			for _, variant := range mi.variants {
				variant.ItemID = it.ID
				if err := st.CreateVariant(ctx, &variant); err != nil {
					return fmt.Errorf("creating variant %q/%q: %w", mi.name, variant.Name, err)
				}
				variants++
			}
		}
	}
	logger.Info("seeded menu", "categories", len(demoMenu), "items", items, "variants", variants)
	return nil
}

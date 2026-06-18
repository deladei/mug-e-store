package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"coffeemug/backend/internal/domain"
	"github.com/lib/pq"
)

// Order mirrors an order header plus its snapshotted lines.
type Order struct {
	ID                 int64             `json:"id"`
	UserID             int64             `json:"user_id"`
	Status             domain.Status     `json:"status"`
	Fulfilment         domain.Fulfilment `json:"fulfilment"`
	Address            string            `json:"address"`
	Phone              string            `json:"phone"`
	SubtotalPesewas    int64             `json:"subtotal_pesewas"`
	DeliveryFeePesewas int64             `json:"delivery_fee_pesewas"`
	DiscountPesewas    int64             `json:"discount_pesewas"`
	TotalPesewas       int64             `json:"total_pesewas"`
	PaystackReference  *string           `json:"paystack_reference,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	Lines              []OrderLine       `json:"lines"`
}

// OrderLine is the immutable snapshot of one purchased line — names and price
// are copied, never foreign-keyed, so menu edits never rewrite history.
type OrderLine struct {
	ID               int64  `json:"id"`
	ItemName         string `json:"item_name"`
	VariantName      string `json:"variant_name"`
	UnitPricePesewas int64  `json:"unit_price_pesewas"`
	Quantity         int    `json:"quantity"`
}

// OrderEvent is one row of the lifecycle audit trail. ActorID is nil when the
// system (the payment webhook) made the change.
type OrderEvent struct {
	ID         int64         `json:"id"`
	OrderID    int64         `json:"order_id"`
	FromStatus domain.Status `json:"from_status"`
	ToStatus   domain.Status `json:"to_status"`
	ActorID    *int64        `json:"actor_id"`
	CreatedAt  time.Time     `json:"created_at"`
}

// LoyaltyEntry is one row of the append-only points ledger.
type LoyaltyEntry struct {
	OrderID   *int64    `json:"order_id"`
	Delta     int64     `json:"delta"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// CheckoutParams carries everything checkout needs. DeliveryFeePesewas is the
// configured flat fee, applied only for delivery; the handler reads it from
// config so the store never hardcodes money.
type CheckoutParams struct {
	UserID             int64
	Fulfilment         domain.Fulfilment
	Address            string
	Phone              string
	DeliveryFeePesewas int64
	IdempotencyKey     string
	// RedeemPoints is how many loyalty points the customer wants to spend on this
	// order (0 = none). The discount and the points actually consumed are decided
	// by domain.Redemption inside the checkout transaction.
	RedeemPoints int64
}

// Checkout turns the user's cart into an order in a single transaction: it
// re-checks availability, snapshots each line, computes the totals in pesewas,
// writes the order, and clears the cart — all atomically. The order is created
// in pending_payment; payment is confirmed later only by the Paystack webhook.
//
// Errors: ErrEmptyCart (no lines), ErrUnavailable (a line went off-menu),
// ErrDuplicateOrder (idempotency key reused).
func (s *Store) Checkout(ctx context.Context, p CheckoutParams) (*Order, error) {
	if !domain.ValidFulfilment(p.Fulfilment) {
		return nil, fmt.Errorf("store: invalid fulfilment %q", p.Fulfilment)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: begin checkout: %w", err)
	}
	defer tx.Rollback()

	cartID, err := s.getOrCreateCartID(ctx, tx, p.UserID)
	if err != nil {
		return nil, err
	}
	view, err := s.cartView(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if len(view.Lines) == 0 {
		return nil, ErrEmptyCart
	}
	for _, l := range view.Lines {
		if !l.Available {
			return nil, ErrUnavailable
		}
	}

	deliveryFee := int64(0)
	if p.Fulfilment == domain.FulfilmentDelivery {
		deliveryFee = p.DeliveryFeePesewas
	}
	subtotal := view.SubtotalPesewas

	// Loyalty redemption. The balance is read INSIDE this transaction, and only
	// after taking a row lock on the user, so two concurrent checkouts cannot
	// both spend the same points (the double-spend trap): the second blocks here
	// until the first commits, then sees the reduced balance. Same FOR UPDATE
	// discipline TransitionOrder uses on the order row.
	discount := int64(0)
	pointsSpent := int64(0)
	if p.RedeemPoints != 0 {
		if _, err := tx.ExecContext(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, p.UserID); err != nil {
			return nil, fmt.Errorf("store: locking user for redemption: %w", err)
		}
		var balance int64
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(delta), 0) FROM loyalty_ledger WHERE user_id = $1`, p.UserID).
			Scan(&balance); err != nil {
			return nil, fmt.Errorf("store: reading loyalty balance: %w", err)
		}
		d, spent, err := domain.Redemption(p.RedeemPoints, balance, subtotal)
		switch {
		case errors.Is(err, domain.ErrInsufficientPoints):
			return nil, ErrInsufficientPoints
		case err != nil:
			return nil, fmt.Errorf("store: redemption: %w", err)
		}
		discount, pointsSpent = d, spent
	}

	total := subtotal + deliveryFee - discount

	order := &Order{
		UserID:             p.UserID,
		Status:             domain.StatusPendingPayment,
		Fulfilment:         p.Fulfilment,
		Address:            p.Address,
		Phone:              p.Phone,
		SubtotalPesewas:    subtotal,
		DeliveryFeePesewas: deliveryFee,
		DiscountPesewas:    discount,
		TotalPesewas:       total,
	}

	const insOrder = `
		INSERT INTO orders (user_id, status, fulfilment, address, phone,
		                    subtotal_pesewas, delivery_fee_pesewas,
		                    discount_pesewas, total_pesewas, idempotency_key)
		VALUES ($1, 'pending_payment', $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	err = tx.QueryRowContext(ctx, insOrder,
		p.UserID, p.Fulfilment, p.Address, p.Phone,
		subtotal, deliveryFee, discount, total, nullString(p.IdempotencyKey),
	).Scan(&order.ID, &order.CreatedAt)
	if c, ok := isUniqueViolation(err); ok && c == "orders_idempotency_key_key" {
		return nil, ErrDuplicateOrder
	}
	if err != nil {
		return nil, fmt.Errorf("store: inserting order: %w", err)
	}

	order.Lines = make([]OrderLine, 0, len(view.Lines))
	const insLine = `
		INSERT INTO order_lines (order_id, item_name, variant_name, unit_price_pesewas, quantity)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`
	for _, l := range view.Lines {
		ol := OrderLine{
			ItemName:         l.ItemName,
			VariantName:      l.VariantName,
			UnitPricePesewas: l.UnitPricePesewas,
			Quantity:         l.Quantity,
		}
		if err := tx.QueryRowContext(ctx, insLine,
			order.ID, ol.ItemName, ol.VariantName, ol.UnitPricePesewas, ol.Quantity).
			Scan(&ol.ID); err != nil {
			return nil, fmt.Errorf("store: inserting order line: %w", err)
		}
		order.Lines = append(order.Lines, ol)
	}

	// The redemption spend is a negative ledger row written in the SAME
	// transaction as the order, so the points are debited atomically with the
	// order's creation — there is no window where one exists without the other.
	if pointsSpent > 0 {
		const insRedeem = `
			INSERT INTO loyalty_ledger (user_id, order_id, delta, reason)
			VALUES ($1, $2, $3, 'redeem_at_checkout')`
		if _, err := tx.ExecContext(ctx, insRedeem, p.UserID, order.ID, -pointsSpent); err != nil {
			return nil, fmt.Errorf("store: writing redemption ledger entry: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM cart_lines WHERE cart_id = $1`, cartID); err != nil {
		return nil, fmt.Errorf("store: clearing cart: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit checkout: %w", err)
	}
	return order, nil
}

// SetPaystackReference records the payment reference for an order (after
// initializing the transaction with Paystack). It is the idempotency anchor
// for the webhook.
func (s *Store) SetPaystackReference(ctx context.Context, orderID int64, reference string) error {
	const q = `UPDATE orders SET paystack_reference = $2 WHERE id = $1`
	res, err := s.db.ExecContext(ctx, q, orderID, reference)
	if err != nil {
		return fmt.Errorf("store: setting paystack reference: %w", err)
	}
	return errIfNoRows(res)
}

// GetOrder returns an order (with lines) by id, for staff/admin reads.
func (s *Store) GetOrder(ctx context.Context, id int64) (*Order, error) {
	return s.getOrder(ctx, id, nil)
}

// GetOrderForUser returns an order only if the user owns it. A non-owner gets
// ErrNotFound, never a 403 — ownership failures must not reveal that the order
// exists (security invariant).
func (s *Store) GetOrderForUser(ctx context.Context, id, userID int64) (*Order, error) {
	return s.getOrder(ctx, id, &userID)
}

func (s *Store) getOrder(ctx context.Context, id int64, userID *int64) (*Order, error) {
	q := `
		SELECT id, user_id, status, fulfilment, address, phone, subtotal_pesewas,
		       delivery_fee_pesewas, discount_pesewas, total_pesewas,
		       paystack_reference, created_at
		FROM orders WHERE id = $1`
	args := []any{id}
	if userID != nil {
		args = append(args, *userID)
		q += " AND user_id = $2"
	}
	o, err := scanOrder(s.db.QueryRowContext(ctx, q, args...))
	if err != nil {
		return nil, err
	}
	if err := s.attachOrderLines(ctx, []*Order{o}); err != nil {
		return nil, err
	}
	return o, nil
}

// GetOrderByReference returns an order by its Paystack reference (the webhook
// entry point).
func (s *Store) GetOrderByReference(ctx context.Context, reference string) (*Order, error) {
	const q = `
		SELECT id, user_id, status, fulfilment, address, phone, subtotal_pesewas,
		       delivery_fee_pesewas, discount_pesewas, total_pesewas,
		       paystack_reference, created_at
		FROM orders WHERE paystack_reference = $1`
	o, err := scanOrder(s.db.QueryRowContext(ctx, q, reference))
	if err != nil {
		return nil, err
	}
	if err := s.attachOrderLines(ctx, []*Order{o}); err != nil {
		return nil, err
	}
	return o, nil
}

// ListUserOrders returns the user's orders newest-first, 20 per (1-based) page,
// each with its snapshotted lines.
func (s *Store) ListUserOrders(ctx context.Context, userID int64, page int) ([]*Order, error) {
	if page < 1 {
		page = 1
	}
	const pageSize = 20
	const q = `
		SELECT id, user_id, status, fulfilment, address, phone, subtotal_pesewas,
		       delivery_fee_pesewas, discount_pesewas, total_pesewas,
		       paystack_reference, created_at
		FROM orders WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`
	rows, err := s.db.QueryContext(ctx, q, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("store: listing user orders: %w", err)
	}
	return s.collectOrders(ctx, rows)
}

// ListOrders returns the staff queue oldest-first. When status is non-nil it
// filters to that one column.
func (s *Store) ListOrders(ctx context.Context, status *domain.Status) ([]*Order, error) {
	q := `
		SELECT id, user_id, status, fulfilment, address, phone, subtotal_pesewas,
		       delivery_fee_pesewas, discount_pesewas, total_pesewas,
		       paystack_reference, created_at
		FROM orders`
	var args []any
	if status != nil {
		args = append(args, string(*status))
		q += " WHERE status = $1"
	}
	q += " ORDER BY created_at ASC, id ASC"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: listing orders: %w", err)
	}
	return s.collectOrders(ctx, rows)
}

// TransitionOrder advances an order to `to`, enforcing the domain state machine
// under a row lock so concurrent staff actions cannot race. A no-op (to ==
// current) is an idempotent success with no event written — this is what makes
// retried Paystack webhooks safe. actorID is nil for system transitions (the
// webhook). On the completed transition it credits loyalty points (1 per GHS 1
// of subtotal) in the same transaction, so points and completion never
// disagree.
//
// Errors: ErrNotFound, ErrInvalidTransition (illegal or from a terminal state).
func (s *Store) TransitionOrder(ctx context.Context, orderID int64, to domain.Status, actorID *int64) (*Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: begin transition: %w", err)
	}
	defer tx.Rollback()

	var (
		from       domain.Status
		fulfilment domain.Fulfilment
		userID     int64
		subtotal   int64
		discount   int64
	)
	const lockQ = `
		SELECT status, fulfilment, user_id, subtotal_pesewas, discount_pesewas
		FROM orders WHERE id = $1 FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, lockQ, orderID).
		Scan(&from, &fulfilment, &userID, &subtotal, &discount); {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("store: locking order: %w", err)
	}

	// Idempotent no-op: nothing to write, just return the current order.
	if domain.IsNoOp(from, to) {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("store: commit no-op: %w", err)
		}
		return s.GetOrder(ctx, orderID)
	}

	if err := domain.CanTransition(fulfilment, from, to); err != nil {
		return nil, fmt.Errorf("%w: %s→%s for %s: %v",
			ErrInvalidTransition, from, to, fulfilment, err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, orderID, to); err != nil {
		return nil, fmt.Errorf("store: updating status: %w", err)
	}
	const insEvent = `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id)
		VALUES ($1, $2, $3, $4)`
	if _, err := tx.ExecContext(ctx, insEvent, orderID, from, to, nullInt64(actorID)); err != nil {
		return nil, fmt.Errorf("store: inserting order event: %w", err)
	}

	// Loyalty is earned only on completion: 1 point per whole GHS of subtotal.
	if to == domain.StatusCompleted {
		points := subtotal / 100
		if points > 0 {
			const insLedger = `
				INSERT INTO loyalty_ledger (user_id, order_id, delta, reason)
				VALUES ($1, $2, $3, 'earn_on_completion')`
			if _, err := tx.ExecContext(ctx, insLedger, userID, orderID, points); err != nil {
				return nil, fmt.Errorf("store: crediting loyalty: %w", err)
			}
		}
	}

	// Cancelling an order that redeemed points refunds them with a compensating
	// positive ledger entry (PRD §S1), so points spent on an order that never
	// completes are not lost — this also covers checkout's cancelDangling path
	// when Paystack initialization fails. The points debited at checkout equal
	// the discount in pesewas (1 point = 1 pesewa), so the refund is the stored
	// discount. Cancelled is terminal and the no-op above short-circuits a repeat
	// cancel, so the refund is written at most once.
	if to == domain.StatusCancelled && discount > 0 {
		const insRefund = `
			INSERT INTO loyalty_ledger (user_id, order_id, delta, reason)
			VALUES ($1, $2, $3, 'refund_on_cancel')`
		if _, err := tx.ExecContext(ctx, insRefund, userID, orderID, discount/domain.PesewasPerPoint); err != nil {
			return nil, fmt.Errorf("store: refunding redeemed points: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit transition: %w", err)
	}
	return s.GetOrder(ctx, orderID)
}

// GetOrderEvents returns an order's audit trail, oldest-first.
func (s *Store) GetOrderEvents(ctx context.Context, orderID int64) ([]OrderEvent, error) {
	const q = `
		SELECT id, order_id, from_status, to_status, actor_id, created_at
		FROM order_events WHERE order_id = $1 ORDER BY created_at ASC, id ASC`
	rows, err := s.db.QueryContext(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("store: listing order events: %w", err)
	}
	defer rows.Close()
	events := []OrderEvent{}
	for rows.Next() {
		var e OrderEvent
		var actor sql.NullInt64
		if err := rows.Scan(&e.ID, &e.OrderID, &e.FromStatus, &e.ToStatus, &actor, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scanning order event: %w", err)
		}
		if actor.Valid {
			e.ActorID = &actor.Int64
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// LoyaltyBalance returns the user's point balance as SUM(delta) — there is no
// stored balance column, so it can never drift from its history.
func (s *Store) LoyaltyBalance(ctx context.Context, userID int64) (int64, error) {
	const q = `SELECT COALESCE(SUM(delta), 0) FROM loyalty_ledger WHERE user_id = $1`
	var bal int64
	if err := s.db.QueryRowContext(ctx, q, userID).Scan(&bal); err != nil {
		return 0, fmt.Errorf("store: loyalty balance: %w", err)
	}
	return bal, nil
}

// LoyaltyLedger returns the user's points history, newest-first.
func (s *Store) LoyaltyLedger(ctx context.Context, userID int64) ([]LoyaltyEntry, error) {
	const q = `
		SELECT order_id, delta, reason, created_at
		FROM loyalty_ledger WHERE user_id = $1 ORDER BY created_at DESC, id DESC`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("store: loyalty ledger: %w", err)
	}
	defer rows.Close()
	entries := []LoyaltyEntry{}
	for rows.Next() {
		var e LoyaltyEntry
		var orderID sql.NullInt64
		if err := rows.Scan(&orderID, &e.Delta, &e.Reason, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scanning loyalty entry: %w", err)
		}
		if orderID.Valid {
			e.OrderID = &orderID.Int64
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// collectOrders scans a set of order rows and attaches their lines in one
// follow-up query (no N+1).
func (s *Store) collectOrders(ctx context.Context, rows *sql.Rows) ([]*Order, error) {
	defer rows.Close()
	orders := []*Order{}
	for rows.Next() {
		o, err := scanOrderRows(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.attachOrderLines(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// attachOrderLines loads and attaches the snapshotted lines for the given
// orders in a single query.
func (s *Store) attachOrderLines(ctx context.Context, orders []*Order) error {
	if len(orders) == 0 {
		return nil
	}
	byID := make(map[int64]*Order, len(orders))
	ids := make([]int64, 0, len(orders))
	for _, o := range orders {
		o.Lines = []OrderLine{}
		byID[o.ID] = o
		ids = append(ids, o.ID)
	}
	const q = `
		SELECT order_id, id, item_name, variant_name, unit_price_pesewas, quantity
		FROM order_lines WHERE order_id = ANY($1) ORDER BY id`
	rows, err := s.db.QueryContext(ctx, q, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("store: listing order lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var orderID int64
		var l OrderLine
		if err := rows.Scan(&orderID, &l.ID, &l.ItemName, &l.VariantName, &l.UnitPricePesewas, &l.Quantity); err != nil {
			return fmt.Errorf("store: scanning order line: %w", err)
		}
		if o := byID[orderID]; o != nil {
			o.Lines = append(o.Lines, l)
		}
	}
	return rows.Err()
}

// scanOrder scans one order header from a single-row query.
func scanOrder(row *sql.Row) (*Order, error) {
	var o Order
	var ref sql.NullString
	err := row.Scan(&o.ID, &o.UserID, &o.Status, &o.Fulfilment, &o.Address, &o.Phone,
		&o.SubtotalPesewas, &o.DeliveryFeePesewas, &o.DiscountPesewas, &o.TotalPesewas,
		&ref, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: scanning order: %w", err)
	}
	if ref.Valid {
		o.PaystackReference = &ref.String
	}
	return &o, nil
}

// scanOrderRows scans one order header from a multi-row cursor.
func scanOrderRows(rows *sql.Rows) (*Order, error) {
	var o Order
	var ref sql.NullString
	err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.Fulfilment, &o.Address, &o.Phone,
		&o.SubtotalPesewas, &o.DeliveryFeePesewas, &o.DiscountPesewas, &o.TotalPesewas,
		&ref, &o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: scanning order: %w", err)
	}
	if ref.Valid {
		o.PaystackReference = &ref.String
	}
	return &o, nil
}

// nullString maps "" to a SQL NULL so that multiple orders without an
// idempotency key do not collide on the UNIQUE constraint (NULLs are distinct).
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullInt64 maps a nil actor id to SQL NULL (a system transition).
func nullInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

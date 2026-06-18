package store

import (
	"context"
	"testing"
	"time"
)

// insertOrderAt inserts an order with a chosen created_at, status and total,
// bypassing Checkout so reporting can be tested across days and statuses.
func insertOrderAt(t *testing.T, st *Store, userID int64, createdAt time.Time, status string, total int64) {
	t.Helper()
	const q = `
		INSERT INTO orders (user_id, status, fulfilment, subtotal_pesewas, total_pesewas, created_at)
		VALUES ($1, $2, 'pickup', $3, $3, $4)`
	if _, err := st.db.ExecContext(context.Background(), q, userID, status, total, createdAt); err != nil {
		t.Fatalf("insert order: %v", err)
	}
}

func TestReportSummary(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	u := seedUser(t, st, "report@example.com")

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	// Today: one completed (revenue 1000), one paid (revenue 2000), one cancelled
	// (no revenue), one still pending (no revenue). 4 orders, 2 confirmed, 3000.
	insertOrderAt(t, st, u.ID, today.Add(8*time.Hour), "completed", 1000)
	insertOrderAt(t, st, u.ID, today.Add(9*time.Hour), "paid", 2000)
	insertOrderAt(t, st, u.ID, today.Add(10*time.Hour), "cancelled", 5000)
	insertOrderAt(t, st, u.ID, today.Add(11*time.Hour), "pending_payment", 9000)
	// Yesterday: one completed (revenue 1500). 1 order, 1 confirmed, 1500.
	insertOrderAt(t, st, u.ID, yesterday.Add(12*time.Hour), "completed", 1500)
	// An order outside the 3-day window must not be counted.
	insertOrderAt(t, st, u.ID, today.AddDate(0, 0, -10), "completed", 7777)

	sum, err := st.ReportSummary(ctx, 3)
	if err != nil {
		t.Fatalf("report summary: %v", err)
	}

	// Window is continuous: 3 days (day-2, yesterday, today).
	if len(sum.Daily) != 3 {
		t.Fatalf("daily len = %d, want 3", len(sum.Daily))
	}
	if sum.From != today.AddDate(0, 0, -2).Format(dateLayout) || sum.To != today.Format(dateLayout) {
		t.Errorf("window = %s..%s, unexpected", sum.From, sum.To)
	}

	// Totals exclude the cancelled, the pending, and the out-of-window order.
	if sum.Totals.Orders != 5 {
		t.Errorf("total orders = %d, want 5", sum.Totals.Orders)
	}
	if sum.Totals.PaidOrders != 3 {
		t.Errorf("total paid orders = %d, want 3", sum.Totals.PaidOrders)
	}
	if sum.Totals.RevenuePesewas != 4500 {
		t.Errorf("total revenue = %d, want 4500 (1000+2000+1500)", sum.Totals.RevenuePesewas)
	}

	// The gap day (day-2) is present and zeroed.
	gap := sum.Daily[0]
	if gap.Orders != 0 || gap.RevenuePesewas != 0 {
		t.Errorf("gap day = %+v, want zeroes", gap)
	}
	// Today's slice: 4 orders, 2 confirmed, revenue 3000.
	last := sum.Daily[2]
	if last.Date != today.Format(dateLayout) {
		t.Fatalf("last day = %s, want today", last.Date)
	}
	if last.Orders != 4 || last.PaidOrders != 2 || last.RevenuePesewas != 3000 {
		t.Errorf("today = %+v, want orders 4 / paid 2 / revenue 3000", last)
	}
}

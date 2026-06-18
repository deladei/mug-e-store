package store

import (
	"context"
	"fmt"
	"time"
)

// DailyStat is one day's slice of the admin summary. Revenue counts only orders
// whose payment was confirmed — anything past pending that was not cancelled.
type DailyStat struct {
	Date           string `json:"date"` // YYYY-MM-DD
	Orders         int64  `json:"orders"`
	PaidOrders     int64  `json:"paid_orders"`
	RevenuePesewas int64  `json:"revenue_pesewas"`
}

// ReportSummary is the admin dashboard's orders/day + revenue/day report over a
// window ending today. Daily is continuous (gap days are zero-filled) so a chart
// can plot it without holes.
type ReportSummary struct {
	From   string      `json:"from"` // YYYY-MM-DD, inclusive
	To     string      `json:"to"`   // YYYY-MM-DD, inclusive (today)
	Totals DailyStat   `json:"totals"`
	Daily  []DailyStat `json:"daily"`
}

const dateLayout = "2006-01-02"

// reportConfirmedStatuses is the SQL predicate for "money actually taken": an
// order counts toward revenue once payment is confirmed (past pending) and it
// was not cancelled. Kept in one place so orders/day and revenue/day agree.
const reportConfirmedFilter = `status NOT IN ('pending_payment', 'cancelled')`

// ReportSummary aggregates orders by calendar day over the last `days` days
// (inclusive of today), counting all orders placed and the confirmed-revenue
// subset. The shop operates in Ghana (GMT/UTC+0), so grouping on the stored
// timestamp's date needs no timezone conversion.
func (s *Store) ReportSummary(ctx context.Context, days int) (*ReportSummary, error) {
	if days < 1 {
		days = 1
	}
	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, -(days - 1))

	const q = `
		SELECT created_at::date AS day,
		       COUNT(*) AS orders,
		       COUNT(*) FILTER (WHERE ` + reportConfirmedFilter + `) AS paid_orders,
		       COALESCE(SUM(total_pesewas) FILTER (WHERE ` + reportConfirmedFilter + `), 0) AS revenue
		FROM orders
		WHERE created_at >= $1
		GROUP BY day`
	rows, err := s.db.QueryContext(ctx, q, start)
	if err != nil {
		return nil, fmt.Errorf("store: report summary: %w", err)
	}
	defer rows.Close()

	byDay := make(map[string]DailyStat, days)
	for rows.Next() {
		var day time.Time
		var d DailyStat
		if err := rows.Scan(&day, &d.Orders, &d.PaidOrders, &d.RevenuePesewas); err != nil {
			return nil, fmt.Errorf("store: scanning report row: %w", err)
		}
		d.Date = day.Format(dateLayout)
		byDay[d.Date] = d
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: report rows: %w", err)
	}

	summary := &ReportSummary{
		From:  start.Format(dateLayout),
		To:    today.Format(dateLayout),
		Daily: make([]DailyStat, 0, days),
	}
	// Walk every day in the window so the series is continuous, filling gaps with
	// zeros and accumulating the totals as we go.
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format(dateLayout)
		d, ok := byDay[date]
		if !ok {
			d = DailyStat{Date: date}
		}
		summary.Daily = append(summary.Daily, d)
		summary.Totals.Orders += d.Orders
		summary.Totals.PaidOrders += d.PaidOrders
		summary.Totals.RevenuePesewas += d.RevenuePesewas
	}
	summary.Totals.Date = "" // totals are not a single day
	return summary, nil
}

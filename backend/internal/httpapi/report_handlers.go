package httpapi

import (
	"net/http"
	"strconv"
)

const (
	reportDefaultDays = 30
	reportMaxDays     = 365
)

// handleReportSummary returns the admin orders/day + revenue/day summary. The
// window is the last ?days days (default 30, clamped to [1, 365]).
func (s *Server) handleReportSummary(w http.ResponseWriter, r *http.Request) {
	days := reportDefaultDays
	if raw := r.URL.Query().Get("days"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			days = n
		}
	}
	if days > reportMaxDays {
		days = reportMaxDays
	}

	summary, err := s.store.ReportSummary(r.Context(), days)
	if err != nil {
		s.serverError(w, "report summary", err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

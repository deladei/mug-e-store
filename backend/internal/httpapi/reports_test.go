package httpapi

import (
	"net/http"
	"testing"
)

const reportPath = "/api/v1/admin/reports/summary"

// TestReportSummary_RoleGate: the revenue report is admin-only. Admin gets it;
// staff and customers are forbidden; an anonymous caller is unauthorized.
func TestReportSummary_RoleGate(t *testing.T) {
	h := newHarness(t)
	admin := h.createUser(t, "admin@example.com", "admin")
	staff := h.createUser(t, "staff@example.com", "staff")
	customer := h.createUser(t, "cust@example.com", "")

	cases := []struct {
		name   string
		token  string
		status int
	}{
		{"admin allowed", h.token(t, admin), http.StatusOK},
		{"staff forbidden", h.token(t, staff), http.StatusForbidden},
		{"customer forbidden", h.token(t, customer), http.StatusForbidden},
		{"anonymous unauthorized", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := h.request(t, http.MethodGet, reportPath, tc.token, nil)
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.status)
			}
		})
	}
}

// TestReportSummary_Shape checks the admin gets a well-formed, continuous series.
func TestReportSummary_Shape(t *testing.T) {
	h := newHarness(t)
	admin := h.createUser(t, "admin2@example.com", "admin")

	resp := h.request(t, http.MethodGet, reportPath+"?days=7", h.token(t, admin), nil)
	var sum struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Totals struct {
			Orders         int64 `json:"orders"`
			RevenuePesewas int64 `json:"revenue_pesewas"`
		} `json:"totals"`
		Daily []struct {
			Date string `json:"date"`
		} `json:"daily"`
	}
	decodeJSONBody(t, resp, &sum)
	if len(sum.Daily) != 7 {
		t.Errorf("daily len = %d, want 7 (continuous window)", len(sum.Daily))
	}
	if sum.From == "" || sum.To == "" {
		t.Errorf("missing window bounds: from=%q to=%q", sum.From, sum.To)
	}
}
